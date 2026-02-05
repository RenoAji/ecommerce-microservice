package infrastructure

import (
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/coreapi"
	"github.com/midtrans/midtrans-go/snap"
)

type MidtransWrapper struct {
    SnapClient snap.Client
    CoreClient coreapi.Client
}

func NewMidtransClient(serverKey string, clientKey string) *MidtransWrapper {
    // 1. Setup Snap Client (for generating Payment URLs)
    s := snap.Client{}
    s.New(serverKey, midtrans.Sandbox)

    // 2. Setup Core Client (for Checking Status/Webhooks)
    c := coreapi.Client{}
    c.New(serverKey, midtrans.Sandbox)

    return &MidtransWrapper{
        SnapClient: s,
        CoreClient: c,
    }
}