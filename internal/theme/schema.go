package theme

type Theme struct {
	Name    string   `yaml:"name"`
	Variant Variant  `yaml:"variant"`
	FZF     FZFTheme `yaml:"fzf"`
}

type Variant string

const (
	VariantLight Variant = "light"
	VariantDark  Variant = "dark"

	DefaultName = "lattice-dark"
)

type StylePreset string

const (
	StyleDefault StylePreset = "default"
	StyleMinimal StylePreset = "minimal"
	StyleFull    StylePreset = "full"
)

type Layout string

const (
	LayoutDefault     Layout = "default"
	LayoutReverse     Layout = "reverse"
	LayoutReverseList Layout = "reverse-list"
)

type BorderStyle string

const (
	BorderRounded    BorderStyle = "rounded"
	BorderSharp      BorderStyle = "sharp"
	BorderBold       BorderStyle = "bold"
	BorderDouble     BorderStyle = "double"
	BorderDashed     BorderStyle = "dashed"
	BorderBlock      BorderStyle = "block"
	BorderThinBlock  BorderStyle = "thinblock"
	BorderHorizontal BorderStyle = "horizontal"
	BorderVertical   BorderStyle = "vertical"
	BorderLine       BorderStyle = "line"
	BorderTop        BorderStyle = "top"
	BorderBottom     BorderStyle = "bottom"
	BorderLeft       BorderStyle = "left"
	BorderRight      BorderStyle = "right"
	BorderInline     BorderStyle = "inline"
	BorderNone       BorderStyle = "none"
)

type InfoStyle string

const (
	InfoDefault     InfoStyle = "default"
	InfoRight       InfoStyle = "right"
	InfoHidden      InfoStyle = "hidden"
	InfoInline      InfoStyle = "inline"
	InfoInlineRight InfoStyle = "inline-right"
)

type BaseColorScheme string

const (
	BaseColorDark   BaseColorScheme = "dark"
	BaseColorLight  BaseColorScheme = "light"
	BaseColorBase16 BaseColorScheme = "base16"
	BaseColor16     BaseColorScheme = "16"
	BaseColorBW     BaseColorScheme = "bw"
)

type FZFTheme struct {
	Style             StylePreset `yaml:"style,omitempty"`
	Layout            Layout      `yaml:"layout,omitempty"`
	Height            string      `yaml:"height,omitempty"`
	MinHeight         string      `yaml:"min-height,omitempty"`
	Popup             string      `yaml:"popup,omitempty"`
	Margin            string      `yaml:"margin,omitempty"`
	Padding           string      `yaml:"padding,omitempty"`
	Border            BorderStyle `yaml:"border,omitempty"`
	ListBorder        BorderStyle `yaml:"list-border,omitempty"`
	InputBorder       BorderStyle `yaml:"input-border,omitempty"`
	PreviewBorder     BorderStyle `yaml:"preview-border,omitempty"`
	HeaderBorder      BorderStyle `yaml:"header-border,omitempty"`
	HeaderLinesBorder BorderStyle `yaml:"header-lines-border,omitempty"`
	FooterBorder      BorderStyle `yaml:"footer-border,omitempty"`
	BorderLabel       string      `yaml:"border-label,omitempty"`
	BorderLabelPos    string      `yaml:"border-label-pos,omitempty"`
	ListLabel         string      `yaml:"list-label,omitempty"`
	ListLabelPos      string      `yaml:"list-label-pos,omitempty"`
	InputLabel        string      `yaml:"input-label,omitempty"`
	InputLabelPos     string      `yaml:"input-label-pos,omitempty"`
	HeaderLabel       string      `yaml:"header-label,omitempty"`
	HeaderLabelPos    string      `yaml:"header-label-pos,omitempty"`
	FooterLabel       string      `yaml:"footer-label,omitempty"`
	FooterLabelPos    string      `yaml:"footer-label-pos,omitempty"`
	PreviewLabel      string      `yaml:"preview-label,omitempty"`
	PreviewLabelPos   string      `yaml:"preview-label-pos,omitempty"`
	Info              string      `yaml:"info,omitempty"`
	Prompt            string      `yaml:"prompt,omitempty"`
	Ghost             string      `yaml:"ghost,omitempty"`
	Separator         string      `yaml:"separator,omitempty"`
	Pointer           string      `yaml:"pointer,omitempty"`
	Marker            string      `yaml:"marker,omitempty"`
	MarkerMultiLine   string      `yaml:"marker-multi-line,omitempty"`
	Gutter            string      `yaml:"gutter,omitempty"`
	GutterRaw         string      `yaml:"gutter-raw,omitempty"`
	Scrollbar         string      `yaml:"scrollbar,omitempty"`
	Ellipsis          string      `yaml:"ellipsis,omitempty"`
	WrapSign          string      `yaml:"wrap-sign,omitempty"`
	PreviewWrapSign   string      `yaml:"preview-wrap-sign,omitempty"`
	PreviewWindow     string      `yaml:"preview-window,omitempty"`
	Color             ColorTheme  `yaml:"color,omitempty"`
}

type ColorTheme struct {
	Base    BaseColorScheme   `yaml:"base,omitempty"`
	Entries map[string]string `yaml:"entries,omitempty"`
}
