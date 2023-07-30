# parse-arin

## Installation

Run the following command, it requires `make` and `golang`:

```sh
make && sudo make install
```

Optionally run tests:

```sh
make test
```

## Usage

Parse directory including test file:

```sh
parse-arin --test-file ./test_silverstar.json --target-dir ./.cache/arin-rir/
```
