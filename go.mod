module maunium.net/go/gomuks

go 1.12

require (
	github.com/disintegration/imaging v1.6.0
	github.com/kyokomi/emoji v2.1.0+incompatible
	github.com/lithammer/fuzzysearch v1.0.2
	github.com/lucasb-eyer/go-colorful v1.0.1
	github.com/mattn/go-runewidth v0.0.4
	github.com/russross/blackfriday/v2 v2.0.1
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	golang.org/x/image v0.0.0-20190321063152-3fc05d484e9f
	golang.org/x/net v0.0.0-20190326090315-15845e8f865b
	gopkg.in/yaml.v2 v2.2.2
	maunium.net/go/mautrix v0.1.0-alpha.3.0.20190326200501-48bbec641543
	maunium.net/go/mauview v0.0.0-20190325223341-4c387be4b686
	maunium.net/go/tcell v0.0.0-20190111223412-5e74142cb009
)

replace maunium.net/go/mautrix => ../mautrix-go

replace maunium.net/go/tcell => ../../Go/tcell

replace maunium.net/go/mauview => ../../Go/mauview
