package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/sprig/v3"
	"github.com/mlabbe/treasure-map/textfunc"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

var version = "github.com/mlabbe/go-template-cli"

// always strict
var FatalMissingInclude = true
var TemplateOptions = "missingkey=error"

// the state of the program
type state struct {
	// options
	defaultTemplateName string
	globs               []string
	templateName        string
	decoder             decoder
	noNewline           bool
	showVersion         bool
	preservePreamble    bool
	outputFilename      string
	varsFilenames       []string
	trusted             bool

	// internal state
	flagSet  *pflag.FlagSet
	template *template.Template
}

// create a new cli instance and bind flags to it
// flag.Parse is called on run
func new(fs *pflag.FlagSet) *state {
	if fs == nil {
		fs = pflag.CommandLine
	}

	cli := &state{
		flagSet: fs,
		decoder: decoder{
			decodeToml,
			"toml",
		},
		defaultTemplateName: "_gotpl_default",
	}

	fs.StringArrayVarP(&cli.globs, "glob", "g", cli.globs, "template file glob. Can be specified multiple times. Make sure not to shell expand the glob.")
	fs.StringVarP(&cli.templateName, "name", "n", cli.templateName, "if specified, execute the template with the given name")
	fs.StringVarP(&cli.outputFilename, "output-file", "o", "", "output filename (outputs to stdout if unspecified)")
	fs.VarP(&cli.decoder, "decoder", "d", "decoder to use for input data. Supported values: json, yaml, toml (default \"toml\")")
	fs.BoolVar(&cli.noNewline, "no-newline", cli.noNewline, "do not print newline at the end of the output")
	fs.BoolVar(&cli.showVersion, "version", cli.showVersion, "show version information and exit")
	fs.BoolVarP(&cli.preservePreamble, "preserve-preamble", "p", cli.preservePreamble, "Preserve build edge specification comments in output file")
	fs.BoolVar(&cli.trusted, "trusted", false, "security: trusted templates -- allow shell and file access to executing machine")

	return cli
}

func (cli *state) replaceOutputWriterFromCli(w io.Writer) (io.Writer, error) {

	if cli.outputFilename == "" {
		return w, nil
	}

	file, err := os.Create(cli.outputFilename)
	if err != nil {
		return nil, err
	}

	return file, err
}

func mergeData(dest, src map[string]any) {
	for k, v := range src {

		if srcMap, ok := v.(map[string]any); ok {

			if destMap, ok := dest[k].(map[string]any); ok {
				// the sub-table exists in both the source and destination already
				mergeData(destMap, srcMap)
			} else {
				dest[k] = srcMap
			}
		} else {
			dest[k] = v
		}
	}
}

// decode all input sources -- all passed-in files, and then any stdin
func (cli *state) decodeAll(r io.Reader) (any, error) {

	mergedData := make(map[string]any)

	for _, varsPath := range cli.varsFilenames {
		file, err := os.Open(varsPath)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		newData, err := cli.decode(file)
		if err != nil {
			return nil, err
		}

		mergeData(mergedData, newData)
	}

	if r != nil {
		// finally, merge any data from stdin
		newData, err := cli.decode(r)
		if err != nil {
			return nil, err
		}

		mergeData(mergedData, newData)
	}

	return mergedData, nil
}

// parse the options and input, decode the input and render the result
func (cli *state) run(args []string, r io.Reader, w io.Writer) (err error) {
	if err := cli.parse(args); err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	if cli.showVersion {
		fmt.Fprintln(w, version)
		return nil
	}

	data, err := cli.decodeAll(r)
	if err != nil {
		return fmt.Errorf("decode all: %w", err)
	}

	preamble := ""
	if cli.preservePreamble {
		if cli.outputFilename == "" {
			return fmt.Errorf("--preserve-preamble specified but output is stdout.  Specify output filename with -o")
		}

		preamble, err = GetPreamble(cli.outputFilename)
		if err != nil {
			return fmt.Errorf("get preamble: %w", err)
		}
	}

	w, err = cli.replaceOutputWriterFromCli(w)
	if err != nil {
		return fmt.Errorf("output file: %w", err)
	}

	fmt.Fprintf(w, "%s", preamble)

	if err := cli.render(w, data); err != nil {
		return fmt.Errorf("render: %w", err)
	}

	return nil
}

func (cli *state) parse(rawArgs []string) error {
	if err := cli.parseFlagset(rawArgs); err != nil {
		return fmt.Errorf("parse raw args: %s", err)
	}

	if _, err := cli.parseFilesAndGlobs(); err != nil {
		return fmt.Errorf("parse opt args: %w", err)
	}

	return nil
}

func (cli *state) parseFlagset(rawArgs []string) error {
	cli.flagSet.SortFlags = false

	if err := cli.flagSet.Parse(rawArgs); err != nil {
		return err
	}

	// for from_file operations, get the relative path from the last specified
	// vars file
	varsFiles := cli.getAllVarsFilesFromArgs()
	relativeDir := ""
	if len(varsFiles) > 0 {
		relativeDir = filepath.Dir(varsFiles[len(varsFiles)-1])
		relativeDir, _ = filepath.Abs(relativeDir) // fixme: clean this up
	}
	fmt.Printf("Relative dir: %s\n", relativeDir)

	cli.template = baseTemplate(cli.defaultTemplateName, cli.trusted, relativeDir)

	return nil
}

// positional args contains zero or more vars files
func (cli *state) getAllVarsFilesFromArgs() []string {

	varsFiles := make([]string, 0)
	for _, arg := range cli.flagSet.Args() {
		if filepath.Ext(arg) == "."+cli.decoder.name {
			varsFiles = append(varsFiles, arg)
		}
	}

	return varsFiles
}

// positional args contains zero or more template files in addition to zero or more globs.
// this returns only the vars files, not the globs
func (cli *state) getAllTemplateFilesFromArgs() []string {
	tplFiles := make([]string, 0)
	for _, arg := range cli.flagSet.Args() {
		if filepath.Ext(arg) != "."+cli.decoder.name {
			tplFiles = append(tplFiles, arg)
		}
	}

	return tplFiles
}

// parse files and globs in the order they were specified, to align with go's
// template engine. should be called after parseFlagset
func (cli *state) parseFilesAndGlobs() (*template.Template, error) {
	var (
		err       error
		globIndex uint8
	)

	cli.flagSet.Visit(func(f *pflag.Flag) {
		switch f.Name {
		case "glob":
			glob := cli.globs[globIndex]
			cli.template, err = cli.template.ParseGlob(glob)
			if err != nil {
				err = fmt.Errorf("error parsing glob %s: %v", glob, err)
				return
			}
			globIndex++
		}
	})

	for _, tplFile := range cli.getAllTemplateFilesFromArgs() {
		cli.template, err = cli.template.ParseFiles(tplFile)
		if err != nil {
			return nil, fmt.Errorf("error parsing file %s: %v", tplFile, err)
		}
	}

	cli.varsFilenames = cli.getAllVarsFilesFromArgs()

	return cli.template, err
}

// decode the input stream into context data
func (cli *state) decode(r io.Reader) (map[string]any, error) {
	if r == nil || cli.decoder.fn == nil {
		return nil, nil
	}

	data, err := cli.decoder.fn(r)
	return data, err
}

type decoderFunc func(in io.Reader) (map[string]any, error)

type decoder struct {
	fn   decoderFunc
	name string
}

//type decoder func(in io.Reader) (map[string]any, error)

func (dec decoder) String() string {
	return dec.name
}

func (dec *decoder) Type() string { return "func" }

func (dec *decoder) Set(kind string) error {
	switch kind {
	case "json":
		dec.fn = decodeJson
	case "yaml":
		dec.fn = decodeYaml
	case "toml":
		dec.fn = decodeToml
	default:
		return fmt.Errorf("unsupported decoder %q", kind)
	}
	return nil
}

func decodeYaml(in io.Reader) (map[string]any, error) {
	var data map[string]any

	dec := yaml.NewDecoder(in)
	for {
		err := dec.Decode(&data)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}
	return data, nil
}

func decodeToml(in io.Reader) (map[string]any, error) {
	var data map[string]any

	dec := toml.NewDecoder(in)

	_, err := dec.Decode(&data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func decodeJson(in io.Reader) (map[string]any, error) {
	var data map[string]any

	dec := json.NewDecoder(in)
	for {
		err := dec.Decode(&data)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}
	return data, nil
}

// render a template
func (cli *state) render(w io.Writer, data any) error {
	templateName, err := cli.selectTemplate()
	if err != nil {
		return fmt.Errorf("select template: %w", err)
	}

	if err := cli.template.ExecuteTemplate(w, templateName, data); err != nil {
		return fmt.Errorf("execute template: %v", err)
	}

	if !cli.noNewline {
		fmt.Fprintln(w)
	}

	return nil
}

func (cli *state) dumpAllTemplateNames() string {
	s := ""

	for i, t := range cli.template.Templates() {
		s += fmt.Sprintf(s, "%d: '%s'\n", i, t.Name())
	}

	return s
}

func (cli *state) selectTemplate() (string, error) {
	templates := cli.template.Templates()

	if len(templates) == 0 {
		return "", errors.New("no templates found")
	}

	// cli --name sets this
	if cli.templateName != "" {
		return cli.templateName, nil
	}

	// if --name not specified, the first template is it
	if len(templates) > 0 {
		return templates[0].Name(), nil
	}

	return "", fmt.Errorf("the --name flag is required when multiple templates are defined and no default template exists.  Existing template names:\n%s", cli.dumpAllTemplateNames())
}

// construct a base templates with custom functions attached
func baseTemplate(defaultName string, trusted bool, relativeDir string) *template.Template {

	tpl := template.New(defaultName)
	tpl = tpl.Option(TemplateOptions)
	tpl = tpl.Funcs(textfunc.MapClosure(sprig.TxtFuncMap(), tpl, FatalMissingInclude))

	tpl = tpl.Funcs(template.FuncMap{
		"from_file": func(path string) string {
			if !trusted {
				fmt.Fprintf(os.Stderr, "fatal: from_file called, but '--trusted' mode not enabled\n")
				os.Exit(1)
			}

			// open this from the relative path of the template, then revert the cwd
			cwd, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "fatal: could not get cwd")
				os.Exit(1)
			}
			defer os.Chdir(cwd)
			os.Chdir(relativeDir)

			bytes, err := os.ReadFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "fatal: from_file failed to read file '%s': %q\n", path, err)
				cwdInError, _ := os.Getwd()
				fmt.Fprintf(os.Stderr, "current working directory: '%s'\n", cwdInError)
				os.Exit(1)
			}

			return string(bytes)
		}})

	return tpl
}
