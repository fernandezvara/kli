# kli Type Conversion TDD Test Suite

Tests all `Get[T]()` type conversions across all configuration sources.

## Usage

```bash
cd examples/conversions
go run main.go
```

## What It Tests

**7 types** x **4 sources** = **28 test cases**

### Sources
- **DEFAULT** - only default values, no env/flag/file
- **ENV** - environment variables override defaults
- **FLAG** - command-line flags override defaults
- **FILE** - JSON config file override defaults

### Types Covered
| Category | Types |
|----------|-------|
| Basic | `string`, `bool` |
| Integers | `int64` |
| Floats | `float64` |
| Time | `time.Duration` |
| Slices | `[]string`, `[]int64` |

## Output Format

```
Source  | Key                       | Expected Type   | Returned Type   | Value                | OK
--------------------------------------------------------------------------------------------------------------
DEFAULT | TEST_STRING               | string          | string          | default_string       | OK
ENV     | TEST_INT64                | int64           | int64           | 42                   | OK
FLAG    | TEST_INT64_SLICE          | []int64         | []int64         | [1000 2000 3000]     | OK
```

## Exit Code

- `0` - all tests passed
- `1` - one or more tests failed


