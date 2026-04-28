---
title: Test reporters
description: Control how bunpy test displays results with built-in and custom reporters.
weight: 5
---

```bash
bunpy test --reporter <name>
```

## Description

A reporter controls how test results are displayed as tests run. bunpy ships three built-in reporters suited to different contexts: a rich console reporter for local development, a compact dot reporter for high-volume test suites, and a JUnit XML reporter for CI systems that parse structured output.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--reporter <name>` | `console` | Reporter to use: `console`, `dot`, `junit` |
| `--reporter-output <path>` | stdout | Write reporter output to a file (useful for `junit`) |
| `--help`, `-h` | | Print help |

## Console reporter (default)

The default reporter prints each test result on its own line as it completes, with timing and a summary at the end.

```bash
bunpy test
```

```
✓ auth › login with valid credentials (2.1 ms)
✓ auth › login with invalid password (0.9 ms)
✗ auth › login with expired token (3.4 ms)
  AssertionError: expected 401, got 200
    at tests/test_auth.py:34

✓ user › create new user (1.8 ms)
✓ user › delete user (0.7 ms)

5 tests, 4 passed, 1 failed (12 ms)
```

Failing tests print the assertion message and file location immediately below the test name, so you can navigate to the failure without scrolling through a wall of output.

The console reporter uses color (red for failures, green for passes) when stdout is a TTY. Colors are disabled automatically when stdout is redirected.

## Dot reporter

The dot reporter prints one character per test: `.` for pass, `F` for failure, `s` for skip. It is compact and suited to suites with hundreds of tests where per-test lines would scroll past too quickly to read.

```bash
bunpy test --reporter dot
```

```
...F....s...........F.......

28 tests, 26 passed, 2 failed, 1 skipped (341 ms)

Failures:
  1) auth › login with expired token (3.4 ms)
     AssertionError: expected 401, got 200
       at tests/test_auth.py:34

  2) billing › charge declined card (5.1 ms)
     AssertionError: expected "declined", got None
       at tests/test_billing.py:88
```

Failures are collected and printed together after all tests finish.

## JUnit XML reporter

The JUnit XML reporter writes a `<testsuites>` XML document that CI systems - Jenkins, GitHub Actions test summary, GitLab test reports, CircleCI - can parse to display test results in a structured UI.

```bash
bunpy test --reporter junit --reporter-output test-results.xml
```

Output file `test-results.xml`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<testsuites tests="5" failures="1" errors="0" time="0.012">
  <testsuite name="auth" tests="3" failures="1" time="0.006">
    <testcase name="login with valid credentials" classname="tests.test_auth" time="0.0021">
    </testcase>
    <testcase name="login with invalid password" classname="tests.test_auth" time="0.0009">
    </testcase>
    <testcase name="login with expired token" classname="tests.test_auth" time="0.0034">
      <failure message="AssertionError: expected 401, got 200">
        Traceback (most recent call last):
          File "tests/test_auth.py", line 34, in _
            expect(resp.status_code).to_be(401)
        AssertionError: expected 401, got 200
      </failure>
    </testcase>
  </testsuite>
  <testsuite name="user" tests="2" failures="0" time="0.006">
    <testcase name="create new user" classname="tests.test_user" time="0.0018">
    </testcase>
    <testcase name="delete user" classname="tests.test_user" time="0.0007">
    </testcase>
  </testsuite>
</testsuites>
```

Each `<testsuite>` corresponds to a test file. Each `<testcase>` corresponds to a `@test`-decorated function. Failures include the full traceback in the `<failure>` element.

## Using JUnit output in GitHub Actions

GitHub Actions can display test results directly in the pull request UI if you upload the JUnit XML as an artifact and use a test-report action:

```yaml
# .github/workflows/ci.yml
- name: Run tests
  run: bunpy test --reporter junit --reporter-output test-results.xml

- name: Upload test results
  uses: actions/upload-artifact@v4
  if: always()   # upload even if tests failed
  with:
    name: test-results
    path: test-results.xml

- name: Publish test report
  uses: mikepenz/action-junit-report@v4
  if: always()
  with:
    report_paths: test-results.xml
    check_name: Test results
```

The `mikepenz/action-junit-report` action posts a check run on the pull request with a table of passed and failed tests, and annotates the diff with failure locations.

## Combining reporters

Run two reporters in one pass by passing `--reporter` twice - one for the terminal, one for CI:

```bash
bunpy test --reporter console --reporter junit --reporter-output test-results.xml
```

The console output goes to stdout and the XML goes to the file. This is useful in local development when you want human-readable output but also want to inspect the XML for debugging a CI-specific failure.

## Reporter output in watch mode

`--watch` always uses the console reporter regardless of `--reporter`. The interactive watcher needs real-time feedback per test; JUnit and dot reporters are suited for batch runs only.
