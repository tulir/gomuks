package abstract

import (
	"go.mau.fi/mauview"

	"go.mau.fi/gomuks/pkg/gomuks"
	"go.mau.fi/gomuks/pkg/rpc"
)

type App interface {
	Gmx() *gomuks.Gomuks
	Rpc() *rpc.GomuksRPC
	App() *mauview.Application
}
