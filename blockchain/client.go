
package blockchain

import (
    "fmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric-sdk-go/api/apitxn/chclient"
)

//the purpose of this struct is basically the same as FabricSetup, but all of the admin information is stripped
type ClientSetup struct {
    ChainCodeID string
    UserName    string
	client          chclient.ChannelClient
}

func (setup *FabricSetup) InitializeUser(username string) (ClientSetup, error) {
    var newClient ClientSetup
    var err error
	newClient.client, err = setup.sdk.NewClient(fabsdk.WithUser(username)).Channel(setup.ChannelID)
    if err != nil {
        return ClientSetup{}, fmt.Errorf("Couldn't create new client for user {%s}: %v", username, err.Error())
    }
    newClient.UserName = username
    newClient.ChainCodeID = setup.ChainCodeID
    return newClient, nil
}


