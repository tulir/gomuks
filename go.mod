module maunium.net/go/gomuks

go 1.14

require (
	github.com/alecthomas/chroma v0.8.0
	github.com/disintegration/imaging v1.6.2
	github.com/gabriel-vasile/mimetype v1.1.1
	github.com/kyokomi/emoji v2.2.2+incompatible
	github.com/lithammer/fuzzysearch v1.1.0
	github.com/lucasb-eyer/go-colorful v1.0.3
	github.com/mattn/go-runewidth v0.0.9
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d // indirect
	github.com/petermattis/goid v0.0.0-20180202154549-b0b1615b78e5 // indirect
	github.com/pkg/errors v0.9.1
	github.com/rivo/uniseg v0.1.0
	github.com/russross/blackfriday/v2 v2.0.1
	github.com/sasha-s/go-deadlock v0.2.0
	github.com/zyedidia/clipboard v0.0.0-20200421031010-7c45b8673834
	go.etcd.io/bbolt v1.3.4
	golang.org/x/image v0.0.0-20200430140353-33d19683fad8
	golang.org/x/net v0.0.0-20200822124328-c89045814202
	gopkg.in/toast.v1 v1.0.0-20180812000517-0a84660828b2
	gopkg.in/vansante/go-ffprobe.v2 v2.0.2
	gopkg.in/yaml.v2 v2.3.0
	maunium.net/go/mautrix v0.7.6
	maunium.net/go/mauview v0.1.1
	maunium.net/go/tcell v0.2.0
)

//replace maunium.net/go/mautrix => ../mautrix-go
replace maunium.net/go/mautrix => github.com/nikofil/mautrix-go v0.5.2-0.20200911232449-6010305aed05
