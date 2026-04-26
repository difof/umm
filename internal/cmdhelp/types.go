package cmdhelp

type Document struct {
	Title    string
	Intro    []string
	Example  *Example
	Sections []Section
}

type Example struct {
	Title string
	Lines []string
}

type Section struct {
	Title  string
	Body   []string
	Code   []string
	Extras []LabelLine
	Fields []Field
}

type Field struct {
	Path   string
	What   string
	Values string
	Extras []LabelLine
}

type LabelLine struct {
	Label string
	Text  string
}
