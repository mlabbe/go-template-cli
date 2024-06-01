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
 - Call `--input-vars` multiple times to merge multiple variable files (see merging variables)
 
As these changes are use-case driven, the fork is considered permanent.

Note that the docs and tests may break as these changes have not necessarily updated these components thoroughly.

### Fork Roadmap ###

 - fix partially generated templates when an error occurs in the outfile case; make generation atomic.

## Usage

    # glob in all of the tpl files.
    # Note this is single quoted -- this is NOT a shell glob.
    tpl --glob '*.tpl' < vars.toml
    
## Templates

The default templates name is `_gotpl_default` and positional arguments are parsed into this root template. That means while its possible to specify multiple arguments, they will overwrite each other unless they use the `define` keyword to define a named template that can be referenced later when executing the template. If a named template is parsed multiple times, the last one will override the previous ones.

Templates from the flag `--glob` are parsed in the order they are specified. So the override rules of the text/template package apply. If a file with the same name is specified multiple times, the last one wins. Even if they are in different directories.

The behavior of the cli tries to stay consistent with the actual behavior of the go template engine.

If the default template exists it will be used unless the `--name` flag is specified. If no default template exists because no positional argument has been provided, the template with the given file name is used, as long as only one file has been parsed. If multiple files have been parsed, the `--name` flag is required to avoid ambiguity.

```bash
tpl '{{ . }}' foo.tpl --glob 'templates/*.tpl'         # default will be used
tpl foo.tpl                                            # foo.tpl will be used
tpl foo.tpl --glob 'templates/*.tpl' --name foo.tpl    # the --name flag is required to select a template by name
```

The ability to parse multiple templates makes sense when defining helper snippets and other named templates to reference using the builtin `template` keyword or the custom `include` function which can be used in pipelines.

note globs need to quotes to avoid shell expansion.

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

    tpl --input-vars first.toml --input-vars second.toml < third.toml
    
In this case, there are three input variable files.  The merge policy is:

 - The leftmost file from the cli is loaded first (lowest key precedence)
 - All files from the cli are loaded in specified sequence
 - Stdin is loaded last, and has the highest key precedence
 - If a key is found from two sources, the later one takes precedence
 - If a table is found in two sources, the keys from both are merged,
   with the later one taking precedence on any conflictns
