package blockchain

import (
	"fmt"
	"github.com/hyperledger/fabric-sdk-go/api/apitxn/chclient"
	chmgmt "github.com/hyperledger/fabric-sdk-go/api/apitxn/chmgmtclient"
	resmgmt "github.com/hyperledger/fabric-sdk-go/api/apitxn/resmgmtclient"
	"github.com/hyperledger/fabric-sdk-go/pkg/config"
	packager "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/ccpackager/gopackager"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/common/cauthdsl"
	"time"
   )

// FabricSetup implementation
type FabricSetup struct {
	ConfigFile      string
	OrgID           string
	ChannelID       string
	ChainCodeID     string
	initialized     bool
	ChannelConfig   string
	ChainCodeGoPath string
	ChainCodePath   string
	OrgAdmin        string
	OrgName         string
	UserName        string
	client          chclient.ChannelClient
	admin           resmgmt.ResourceMgmtClient
	sdk             *fabsdk.FabricSDK
}

// Initialize reads the configuration file and sets up the client, chain and event hub
func (setup *FabricSetup) Initialize() error {

	// Add parameters for the initialization
	if setup.initialized {
		return fmt.Errorf("sdk already initialized")
	}

	// Initialize the SDK with the configuration file
	sdk, err := fabsdk.New(config.FromFile(setup.ConfigFile))
	if err != nil {
		return fmt.Errorf("failed to create sdk: %v", err)
	}
	setup.sdk = sdk

	// Channel management client is responsible for managing channels (create/update channel)
	// Supply user that has privileges to create channel (in this case orderer admin)
	chMgmtClient, err := setup.sdk.NewClient(fabsdk.WithUser(setup.OrgAdmin), fabsdk.WithOrg(setup.OrgName)).ChannelMgmt()
	if err != nil {
		return fmt.Errorf("failed to add Admin user to sdk: %v", err)
	}

	// Org admin user is signing user for creating channel.
	// The session method is the only way for now to get the user identity.
	session, err := setup.sdk.NewClient(fabsdk.WithUser(setup.OrgAdmin), fabsdk.WithOrg(setup.OrgName)).Session()
	if err != nil {
		return fmt.Errorf("failed to get session for %s, %s: %s", setup.OrgName, setup.OrgAdmin, err)
	}
    orgAdminUser := session
    req := chmgmt.SaveChannelRequest{ChannelID: setup.ChannelID, ChannelConfig: setup.ChannelConfig, SigningIdentity: orgAdminUser}
	if err = chMgmtClient.SaveChannel(req); err != nil {
		return fmt.Errorf("failed to create channel: %v", err)
	}

	// Allow orderer to process channel creation
	time.Sleep(time.Second * 5)

	// The resource management client is a client API for managing system resources
	// It will allow us to directly interact with the blockchain. It can be associated with the admin status
	setup.admin, err = setup.sdk.NewClient(fabsdk.WithUser(setup.OrgAdmin)).ResourceMgmt()
	if err != nil {
		return fmt.Errorf("failed to create new resource management client: %v", err)
	}

	// Org peers join channel
	if err = setup.admin.JoinChannel(setup.ChannelID); err != nil {
		return fmt.Errorf("org peers failed to join the channel: %v", err)
	}
    setup.initialized = true
	fmt.Println("Initialization Successful")
	return nil

}

func (setup *FabricSetup) InstallAndInstantiateCC() error {

	// Create a new go lang chaincode package and initializing it with our chaincode
	ccPkg, err := packager.NewCCPackage(setup.ChainCodePath, setup.ChainCodeGoPath)
	if err != nil {
		return fmt.Errorf("failed to create chaincode package: %v", err)
	}

    version := "3.06"
    //3.0 huh, thats a lot of edits
    //yay 2.0! it doesn't mean anything though
	// Install our chaincode on org peers
	// The resource management client send the chaincode to all peers in its channel in order for them to store it and interact with it later
	installCCReq := resmgmt.InstallCCRequest{Name: setup.ChainCodeID, Path: setup.ChainCodePath, Version: version, Package: ccPkg}
	_, err = setup.admin.InstallCC(installCCReq)
	if err != nil {
		return fmt.Errorf("failed to install cc to org peers %v", err)
	}

	// Set up chaincode policy
	// The chaincode policy is required if your transactions must follow some specific rules
	// If you don't provide any policy every transaction will be endorsed, and it's probably not what you want
	// In this case, we set the rule to : Endorse the transaction if the transaction have been signed by a member from the org "org1.samtest.com"
	ccPolicy := cauthdsl.SignedByAnyMember([]string{"org1.samtest.com"})

	// Instantiate our chaincode on org peers
	// The resource management client tells to all peers in its channel to instantiate the chaincode previously installed
	err = setup.admin.InstantiateCC(setup.ChannelID, resmgmt.InstantiateCCRequest{Name: setup.ChainCodeID, Path: setup.ChainCodePath, Version: version, Args: [][]byte{[]byte("init")}, Policy: ccPolicy})
	if err != nil {
		return fmt.Errorf("failed to instantiate the chaincode: %v", err)
    }
    // Channel client is used to query and execute transactions
	setup.client, err = setup.sdk.NewClient(fabsdk.WithUser(setup.UserName)).Channel(setup.ChannelID)
	if err != nil {
		return fmt.Errorf("failed to create new channel client: %v", err)
	}
	fmt.Println("Chaincode Installation & Instantiation Successful")   

	return nil
}

func (setup *FabricSetup) swapUser(user string) error {
    //swap the "client" to a different user based on the user passed in
    if user == "Admin" {
        return fmt.Errorf("cannot swap client user to Admin")
    }
    var err error
    oldClient := setup.client
    setup.client, err = setup.sdk.NewClient(fabsdk.WithUser(setup.UserName)).Channel(setup.ChannelID)
    if err != nil {
        setup.client = oldClient
        return fmt.Errorf("couldn't swap to user {%s}: %v", user, err.Error())
    }
    return nil
}

