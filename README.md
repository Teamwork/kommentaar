[![Build Status](https://travis-ci.org/Teamwork/kommentaar.svg?branch=master)](https://travis-ci.org/Teamwork/kommentaar)
[![codecov](https://codecov.io/gh/Teamwork/kommentaar/branch/master/graph/badge.svg)](https://codecov.io/gh/Teamwork/kommentaar)

Generate OpenAPI files from comments in Go files.

The idea is that you can write documentation in your comments in a simple
readable manner.

A simple example:

| Code | Explanation |
| ---- | ----------- |
|
| // POST /foo foobar | X |
| // Create a new foo. | X |
| // | X |
| // This will create a new foo object for a customer. It's important to remember | X |
| // that only Pro customers have access to foos. | X |
| // | X |
| // Form: | X |
| //   subject: The subject {string, required}. | X |
| //   message: The message {string}. | X |
| // | X |
| // Response 200 (application/json): | X |
| //   $object: responseObject | X |
