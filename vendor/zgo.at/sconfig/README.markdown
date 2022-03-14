`sconfig` is a simple and functional configuration file parser for Go.

Import as `zgo.at/sconfig`; API docs: https://godocs.io/zgo.at/sconfig

Go 1.5 and newer should work, but the test suite only runs with 1.7 and newer.

What does it look like?
-----------------------

A file like this:

```apache
# This is a comment

port 8080 # This is also a comment

# Look ma, no quotes!
base-url http://example.com

# We'll parse these in a []*regexp.Regexp
match ^foo.+
match ^b[ao]r

# Two values
order allow deny

host  # Idented lines are collapsed
    arp242.net         # My website
    goatcounter.com    # My other website

address arp242.net
```

Can be parsed with:

```go
package main

import (
    "fmt"
    "os"

    "zgo.at/sconfig"

    // Types that need imports are in handlers/pkgname
    _ "zgo.at/sconfig/handlers/regexp"
)

type Config struct {
    Port    int64
    BaseURL string
    Match   []*regexp.Regexp
    Order   []string
    Hosts   []string
    Address string
}

func main() {
    config := Config{}
    err := sconfig.Parse(&config, "config", sconfig.Handlers{
        // Custom handler
        "address": func(line []string) error {
            addr, err := net.LookupHost(line[0])
            if err != nil {
                return err
            }

            config.Address = addr[0]
            return nil
        },
    })
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error parsing config: %v", err)
        os.Exit(1)
    }

    fmt.Printf("%#v\n", config)
}
```

Will result in:

    example.Config{
        Port:    8080,
        BaseURL: "http://example.com",
        Match:   []*regexp.Regexp{[..], [..]},
        Order:   []string{"allow", "deny"},
        Hosts:   []string{"arp242.net", "goatcounter.com"},
        Address: "arp242.net",
    }

But why not...
--------------

- JSON?<br>
  JSON is [not intended for configuration files][json].
- YAML?<br>
  I don't like the whitespace significance in config files, and [YAML can have
  some weird behaviour][yaml].
- XML?<br>
  It's overly verbose.
- INI or TOML?<br>
  They're both fine, I just don't like the syntax much. Typing all those pesky
  `=` and `"` characters is just so much work man!
- Viper?<br>
  Mostly untyped, quite complex, [a lot of
  dependencies](https://godoc.org/github.com/spf13/viper?import-graph).

Isn't "rolling your own" a bad idea? I don't think so. It's not that hard, and
the syntax is simple/intuitive enough to be grokable by most people.

How do I...
-----------

### Validate fields?

Handlers can be chained. For example the default handler for `int64` is:

    RegisterType("int64", ValidateSingleValue(), handleInt64)

`ValidateSingleValue()` returns a type handler that will give an error if there
isn't a single value for this key; for example this is an error:

    foo 42 42

There are several others as well. See `Validate*()` in godoc. You can add more
complex validation handlers if you want, but in general I would recommend just
using plain ol' `if` statements.

Adding things such as tag-based validation isn't a goal at this point. I'm not
at all that sure this is a common enough problem that needs solving, and there
are already many other packages which do this (no need to reinvent the wheel).

My personal recommendation would be [zvalidate][zvalidate], mostly because I
wrote it ;-)

### Set default values?

Set them before parsing:

    c := MyConfig{Value: "The default"}
    sconfig.Parse(&c, "a-file", nil)

### Override from the environment/flags/etc.?

There is no direct built-in support for that, but there is `Fields()` to list
all the field names. For example:

    c := MyConfig{Foo string}
    sconfig.Parse(&c, "a-file", nil)

    for name, val := range sconfig.Fields(&c) {
        if flag[name] != "" {
            val.SetString(flag[name])
        }
    }

### Use `int` types? I get an error?

Only `int64` and `uint64` are handled by default; this should be fine for almost
all use cases of this package. If you want to add any of the other (u)int types
you can do easily with your own type handler.

"lol, no generics", or something, I guess.

Note that the size of `int` and `uint` are platform-dependent, so adding those
may not be a good idea.

### Use my own types as config fields?

You have three options:

- Add a type handler with `sconfig.RegisterType()`.
- Make your type satisfy the `encoding.TextUnmarshaler` interface.
- Add a `Handler` in `sconfig.Parse()`.

### I get a "don’t know how to set fields of the type ..." error if I try to add a new type handler

Include the package name; even if the type handler is in the same package. Do:

    sconfig.RegisterType("[]main.RecordT", func(v []string) (interface{}, error) { .. }

and not:

    sconfig.RegisterType("[]RecordT", func(v []string) (interface{}, error) { .. }

Replace `main` with the appropriate package name.

Syntax
------

The syntax of the file is very simple.

### Definitions

- Whitespace: any Unicode whitespace (Zs or "Separator, Space" category).
- Hash: `#` (U+0023), Backslash: `\` (U+005C), Space: a space (U+0020), NULL: U+0000
- Newline: LF (U+000A) or CR+LF (U+000D, U+000A).
- Line: Any set of characters ending with a Newline

### Reading the file

- A file must be encoded in UTF-8.

- Everything after the first Hash is considered to be a comment and will be
  ignored unless a Hash is immediately preceded by a Backslash.

- All Whitespace is collapsed to a single Space unless a Whitespace character is
  preceded by a Backslash.

- Any Backslash immediately preceded by a Backslash will be treated as a single
  Backslash.

- Any Backslash immediately followed by anything other than a Hash, Whitespace,
  or Backslash is treated as a single Backslash.

- Anything before the first Whitespace is considered the Key.

  - Any character except Whitespace and NULL bytes are allowed in the Key.
  - The special Key `source` can be used to include other config files. The
    Value for this must be a path. 

- Anything after the first Whitespace is considered the Value.

  - Any character except NULL bytes are allowed in the Value.
  - The Value is optional.

- All Lines that start with one or more Whitespace characters will be appended
  to the last Value, even if there are blank lines or comments in between. The
  leading whitespace will be removed.

Alternatives
------------

Aside from those mentioned in the "But why not..." section above:

- [github.com/kovetskiy/ko](https://github.com/kovetskiy/ko)
- [github.com/stevenroose/gonfig](https://github.com/stevenroose/gonfig)

Probably others? Open an issue/PR and I'll add it.


[json]: http://www.arp242.net/json-config.html
[yaml]: http://www.arp242.net/yaml-config.html
[zvalidate]: https://github.com/arp242/zvalidate

