package bunpy

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildS3 returns the bunpy.s3 module.
func BuildS3(_ *goipyVM.Interp) *goipyObject.Module {
	m := &goipyObject.Module{Name: "bunpy.s3", Dict: goipyObject.NewDict()}
	m.Dict.SetStr("connect", &goipyObject.BuiltinFunc{
		Name: "connect",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			bucket, endpoint, accessKey, secretKey, region := "", "https://s3.amazonaws.com", "", "", ""
			if kwargs != nil {
				if v, ok := kwargs.GetStr("bucket"); ok {
					bucket = v.(*goipyObject.Str).V
				}
				if v, ok := kwargs.GetStr("endpoint"); ok {
					endpoint = v.(*goipyObject.Str).V
				}
				if v, ok := kwargs.GetStr("access_key"); ok {
					accessKey = v.(*goipyObject.Str).V
				}
				if v, ok := kwargs.GetStr("secret_key"); ok {
					secretKey = v.(*goipyObject.Str).V
				}
				if v, ok := kwargs.GetStr("region"); ok {
					region = v.(*goipyObject.Str).V
				}
			}
			if accessKey == "" {
				accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
			}
			if secretKey == "" {
				secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
			}
			if region == "" {
				region = os.Getenv("AWS_REGION")
			}
			if region == "" {
				region = "us-east-1"
			}
			if bucket == "" {
				return nil, fmt.Errorf("bunpy.s3.connect(): bucket is required")
			}
			if accessKey == "" || secretKey == "" {
				return nil, fmt.Errorf("bunpy.s3.connect(): access_key and secret_key are required")
			}
			c := &s3Client{
				bucket:    bucket,
				endpoint:  strings.TrimRight(endpoint, "/"),
				accessKey: accessKey,
				secretKey: secretKey,
				region:    region,
				http:      &http.Client{},
			}
			return buildS3Instance(c), nil
		},
	})
	return m
}

type s3Client struct {
	bucket    string
	endpoint  string
	accessKey string
	secretKey string
	region    string
	http      *http.Client
}

func buildS3Instance(c *s3Client) *goipyObject.Instance {
	cls := &goipyObject.Class{Name: "S3Client", Dict: goipyObject.NewDict()}
	inst := &goipyObject.Instance{Class: cls, Dict: goipyObject.NewDict()}

	set := func(name string, fn func([]goipyObject.Object, *goipyObject.Dict) (goipyObject.Object, error)) {
		inst.Dict.SetStr(name, &goipyObject.BuiltinFunc{Name: name, Call: func(_ any, args []goipyObject.Object, kw *goipyObject.Dict) (goipyObject.Object, error) {
			return fn(args, kw)
		}})
	}

	set("read", func(args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		key := strArg(args, 0)
		req, err := c.newRequest("GET", key, nil, 0)
		if err != nil {
			return nil, err
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("S3 GET %s: %s", key, resp.Status)
		}
		data, _ := io.ReadAll(resp.Body)
		return &goipyObject.Bytes{V: data}, nil
	})

	set("write", func(args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		key := strArg(args, 0)
		var body []byte
		switch v := args[1].(type) {
		case *goipyObject.Str:
			body = []byte(v.V)
		case *goipyObject.Bytes:
			body = v.V
		default:
			return nil, fmt.Errorf("s3.write(): data must be str or bytes")
		}
		req, err := c.newRequest("PUT", key, bytes.NewReader(body), int64(len(body)))
		if err != nil {
			return nil, err
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		resp.Body.Close()
		if resp.StatusCode != 200 && resp.StatusCode != 204 {
			return nil, fmt.Errorf("S3 PUT %s: %s", key, resp.Status)
		}
		return goipyObject.None, nil
	})

	set("delete", func(args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		key := strArg(args, 0)
		req, err := c.newRequest("DELETE", key, nil, 0)
		if err != nil {
			return nil, err
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		resp.Body.Close()
		return goipyObject.None, nil
	})

	set("exists", func(args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		key := strArg(args, 0)
		req, err := c.newRequest("HEAD", key, nil, 0)
		if err != nil {
			return goipyObject.BoolOf(false), nil
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return goipyObject.BoolOf(false), nil
		}
		resp.Body.Close()
		return goipyObject.BoolOf(resp.StatusCode == 200), nil
	})

	set("list", func(_ []goipyObject.Object, kw *goipyObject.Dict) (goipyObject.Object, error) {
		prefix := ""
		if kw != nil {
			if v, ok := kw.GetStr("prefix"); ok {
				if s, ok2 := v.(*goipyObject.Str); ok2 {
					prefix = s.V
				}
			}
		}
		return c.listObjects(prefix)
	})

	set("presign", func(args []goipyObject.Object, kw *goipyObject.Dict) (goipyObject.Object, error) {
		key := strArg(args, 0)
		expires := 3600
		if kw != nil {
			if v, ok := kw.GetStr("expires"); ok {
				if n, ok2 := v.(*goipyObject.Int); ok2 {
					expires = int(n.Int64())
				}
			}
		}
		u, err := c.presignURL("GET", key, expires)
		if err != nil {
			return nil, err
		}
		return &goipyObject.Str{V: u}, nil
	})

	set("close", func(_ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		return goipyObject.None, nil
	})
	set("__enter__", func(_ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		return inst, nil
	})
	set("__exit__", func(_ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		return goipyObject.BoolOf(false), nil
	})

	return inst
}

// objectURL returns the path-style URL for a key.
func (c *s3Client) objectURL(key string) string {
	return c.endpoint + "/" + c.bucket + "/" + strings.TrimPrefix(key, "/")
}

// newRequest builds a signed *http.Request.
func (c *s3Client) newRequest(method, key string, body io.Reader, contentLength int64) (*http.Request, error) {
	rawURL := c.objectURL(key)
	req, err := http.NewRequest(method, rawURL, body)
	if err != nil {
		return nil, err
	}
	req.ContentLength = contentLength

	now := time.Now().UTC()
	dateStr := now.Format("20060102")
	dateTimeStr := now.Format("20060102T150405Z")

	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = io.ReadAll(body)
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.ContentLength = int64(len(bodyBytes))
	}
	bodyHash := sha256Hex(bodyBytes)

	req.Header.Set("x-amz-date", dateTimeStr)
	req.Header.Set("x-amz-content-sha256", bodyHash)
	req.Header.Set("host", req.URL.Host)

	c.signRequest(req, bodyHash, dateStr, dateTimeStr)
	return req, nil
}

func (c *s3Client) signRequest(req *http.Request, bodyHash, dateStr, dateTimeStr string) {
	signedHeaders, canonicalHeaders := buildCanonicalHeaders(req)
	canonicalURI := req.URL.EscapedPath()
	canonicalQuery := req.URL.RawQuery

	canonicalReq := strings.Join([]string{
		req.Method,
		canonicalURI,
		canonicalQuery,
		canonicalHeaders,
		signedHeaders,
		bodyHash,
	}, "\n")

	scope := dateStr + "/" + c.region + "/s3/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + dateTimeStr + "\n" + scope + "\n" + sha256Hex([]byte(canonicalReq))

	signingKey := deriveSigningKey(c.secretKey, dateStr, c.region, "s3")
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	req.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		c.accessKey, scope, signedHeaders, signature,
	))
}

// PresignURL returns a pre-signed URL. Exported for testing.
func (c *s3Client) presignURL(method, key string, expires int) (string, error) {
	now := time.Now().UTC()
	dateStr := now.Format("20060102")
	dateTimeStr := now.Format("20060102T150405Z")
	scope := dateStr + "/" + c.region + "/s3/aws4_request"

	rawURL := c.objectURL(key)
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	q.Set("X-Amz-Credential", c.accessKey+"/"+scope)
	q.Set("X-Amz-Date", dateTimeStr)
	q.Set("X-Amz-Expires", strconv.Itoa(expires))
	q.Set("X-Amz-SignedHeaders", "host")
	u.RawQuery = q.Encode()

	canonicalReq := strings.Join([]string{
		method,
		u.EscapedPath(),
		u.RawQuery,
		"host:" + u.Host + "\n",
		"host",
		"UNSIGNED-PAYLOAD",
	}, "\n")

	stringToSign := "AWS4-HMAC-SHA256\n" + dateTimeStr + "\n" + scope + "\n" + sha256Hex([]byte(canonicalReq))
	signingKey := deriveSigningKey(c.secretKey, dateStr, c.region, "s3")
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	q.Set("X-Amz-Signature", signature)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (c *s3Client) listObjects(prefix string) (goipyObject.Object, error) {
	rawURL := c.endpoint + "/" + c.bucket + "?list-type=2"
	if prefix != "" {
		rawURL += "&prefix=" + url.QueryEscape(prefix)
	}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	dateStr := now.Format("20060102")
	dateTimeStr := now.Format("20060102T150405Z")
	bodyHash := sha256Hex(nil)
	req.Header.Set("x-amz-date", dateTimeStr)
	req.Header.Set("x-amz-content-sha256", bodyHash)
	req.Header.Set("host", req.URL.Host)
	c.signRequest(req, bodyHash, dateStr, dateTimeStr)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("S3 list: %s", resp.Status)
	}
	body, _ := io.ReadAll(resp.Body)

	type contents struct {
		Key string `xml:"Key"`
	}
	type result struct {
		Contents []contents `xml:"Contents"`
	}
	var r result
	if err2 := xml.Unmarshal(body, &r); err2 != nil {
		return nil, fmt.Errorf("S3 list parse: %w", err2)
	}

	items := make([]goipyObject.Object, len(r.Contents))
	for i, c2 := range r.Contents {
		items[i] = &goipyObject.Str{V: c2.Key}
	}
	return &goipyObject.List{V: items}, nil
}

func buildCanonicalHeaders(req *http.Request) (signedHeaders, canonicalHeaders string) {
	headers := make(map[string]string)
	for k, v := range req.Header {
		headers[strings.ToLower(k)] = strings.Join(v, ",")
	}
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k + ":" + headers[k] + "\n")
	}
	return strings.Join(keys, ";"), sb.String()
}

// DeriveSigningKey is exported for testing.
func DeriveSigningKey(secret, date, region, service string) []byte {
	return deriveSigningKey(secret, date, region, service)
}

func deriveSigningKey(secret, date, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(date))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte("aws4_request"))
}

// HMACSHA256 is exported for testing.
func HMACSHA256(key, data []byte) []byte { return hmacSHA256(key, data) }

// SHA256Hex is exported for testing.
func SHA256Hex(data []byte) string { return sha256Hex(data) }

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
