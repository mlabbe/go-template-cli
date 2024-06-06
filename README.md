# Go Template CLI (tpl)

Render json, yaml, & toml with go templates from the command line.

The templates are executed with the [text/template](https://pkg.go.dev/text/template) package. This means they come with the additional risks and benefits of the text template engine.

## Fork Status ##

This is a fork of https://github.com/bluebrown/go-template-cli.  It contains the following changes:

 - Calling `include` on a template name that doesn't exist fails. Previously it silently failed.
 - Default decoder is toml instead of json
 - Templates missing variables immediately error out
 - Template --options option removed
 - New optional `--output-file` argument writes to a file instead of relying on piping
 - New option `--preserve-preamble` preserves build edge specification in output file header
 - Remove `--file` option for templates, use positional arguments instead.
 - Previous positional arguments allowing for templates in command line arguments removed.
 - Positional arguments are now either vars files, or template files, depending on extension.
 - The first template name passed in or globbed is the default, unless overridden by `--name`
 - Add new `from_file` template function that inserts the contents of a file in the template
 - Add new `--trusted` flag that limits template functions that can cause harm (see security limitations)
 - In the case of an error while rendering, output files are no longer partially overwritten
 
As these changes are use-case driven, the fork is considered permanent.

Note that the docs and tests may break as these changes have not necessarily updated these components thoroughly.

### Fork Roadmap ###

 - fix partially generated templates when an error occurs in the outfile case; make generation atomic.
 - support alt braces `[_ _]`

## Usage

    # glob in all of the tpl files.
    # Note this is single quoted -- this is NOT a shell glob.
    tpl --glob '*.tpl' < vars.toml
    
    # Specify two vars files and one template file
    tpl var0.toml var1.toml index.tpl
    
## Decoders

By default input data is decoded as toml and passed to the template to execute. It is possible to use an alternative decoder. The supported decoders are:

- json
- yaml
- toml

## Functions

Next to the builtin functions, [Sprig functions](http://masterminds.github.io/sprig/) and [treasure-map functions](https://github.com/mlabbe/treasure-map) are available.

## Installation

### Go

If you have go installed, you can use the `go install` command to install the binary.

```bash
go install github.com/mlabbe/go-template-cli/cmd/tpl@latest
```

## Merging Variables ##

It is possible to have multiple input variable files.  Consider:

    tpl first.toml second.toml < third.toml
    
In this case, there are three input variable files.  The merge policy is:

 - The leftmost file from the cli is loaded first (lowest key precedence)
 - All files from the cli are loaded in specified sequence
 - Stdin is loaded last, and has the highest key precedence
 - If a key is found from two sources, the later one takes precedence
 - If a table is found in two sources, the keys from both are merged,
   with the later one taking precedence on any conflictns

## Security Limitations ##

Templates have access to function that can leak system state in a manner that may lead to a breach.  For example, Sprig supplies functions to [read the environment](https://masterminds.github.io/sprig/os.html).

The new `from_file` allows any file in the system to be read into a template.  This is disabled unless `--trusted` is passed in.

The bottom line - don't execute arbitrary templates from untrusted sources.
