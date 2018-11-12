package blockchain

import (
	"fmt"
    "path"
    "encoding/pem"
    "crypto/x509"
	"github.com/hyperledger/fabric-sdk-go/api/apitxn/chclient"
	chmgmt "github.com/hyperledger/fabric-sdk-go/api/apitxn/chmgmtclient"
	resmgmt "github.com/hyperledger/fabric-sdk-go/api/apitxn/resmgmtclient"
	"github.com/hyperledger/fabric-sdk-go/pkg/config"
	packager "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/ccpackager/gopackager"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/common/cauthdsl"
	"time"

	ca "github.com/hyperledger/fabric-sdk-go/api/apifabca"
	//apiconfig "github.com/hyperledger/fabric-sdk-go/api/apiconfig"
    "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client"
	fabricCAClient "github.com/hyperledger/fabric-sdk-go/pkg/fabric-ca-client"
	cryptosuite "github.com/hyperledger/fabric-sdk-go/pkg/cryptosuite/bccsp/sw"
	kvs "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/keyvaluestore"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/identity"
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

    version := "2.00"
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
	// In this case, we set the rule to : Endorse the transaction if the transaction have been signed by a member from the org "org1.hf.chainhero.io"
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

//i have no idea which version of fabric sdk this works for
func (setup *FabricSetup) NewUser(userName string, org string) error {
    cfg, err := config.FromFile(setup.ConfigFile)()
    if err != nil {
        return fmt.Errorf("couldn't get config: %v", err)
    }
    mspID, err := cfg.MspID("Org1")
    if err != nil {
        return fmt.Errorf("mspID error: %v", err)
    }
    caConfig, err := cfg.CAConfig("Org1")
    if err != nil {
        return fmt.Errorf("CAConfig error: %v", err)
    }
    client := fabricclient.NewClient(cfg)
    csp, err := cryptosuite.GetSuiteByConfig(cfg)
    if err != nil {
        return fmt.Errorf("cryptosuite error: %v", err)
    }
    stateStorePath := "/tmp/enroll_user"
    client.SetCryptoSuite(csp)


	stateStore, err := kvs.NewFileKeyValueStore(&kvs.FileKeyValueStoreOptions{
		Path: stateStorePath,
		KeySerializer: func(key interface{}) (string, error) {
			keyString, ok := key.(string)
			if !ok {
				return "", fmt.Errorf("converting key to string failed")
			}
			return path.Join(stateStorePath, keyString+".json"), nil
		},
	})
	if err != nil {
		return fmt.Errorf("CreateNewFileKeyValueStore return error[%s]", err)
	}
    client.SetStateStore(stateStore)

	caClient, err := fabricCAClient.NewFabricCAClient("Org1", cfg, csp)
	if err != nil {
		return fmt.Errorf("NewFabricCAClient return error: %v", err)
	}

	// Admin user is used to register, enroll and revoke a test user
	adminUser, err := client.LoadUserFromStateStore("admin")

	if err != nil {
		return fmt.Errorf("client.LoadUserFromStateStore return error: %v", err)
	}
	if adminUser == nil {
		key, cert, err := caClient.Enroll("admin", "adminpw")
		if err != nil {
			return fmt.Errorf("Enroll return error: %v", err)
		}
		if key == nil {
			return fmt.Errorf("private key return from Enroll is nil")
		}
		if cert == nil {
			return fmt.Errorf("cert return from Enroll is nil")
		}

		certPem, _ := pem.Decode(cert)
		if certPem == nil {
			return fmt.Errorf("Fail to decode pem block")
		}

		cert509, err := x509.ParseCertificate(certPem.Bytes)
		if err != nil {
			return fmt.Errorf("x509 ParseCertificate return error: %v", err)
		}
		if cert509.Subject.CommonName != "admin" {
			return fmt.Errorf("CommonName in x509 cert is not the enrollmentID")
		}
		adminUser2 := identity.NewUser("admin", mspID)
		adminUser2.SetPrivateKey(key)
		adminUser2.SetEnrollmentCertificate(cert)
		err = client.SaveUserToStateStore(adminUser2)
		if err != nil {
			return fmt.Errorf("client.SaveUserToStateStore return error: %v", err)
		}
		adminUser, err = client.LoadUserFromStateStore("admin")
		if err != nil {
			return fmt.Errorf("client.LoadUserFromStateStore return error: %v", err)
		}
		if adminUser == nil {
			return fmt.Errorf("client.LoadUserFromStateStore return nil")
		}
	}

	registerRequest := ca.RegistrationRequest{
		Name:        userName,
		Type:        "user",
		Affiliation: "Org1.samtest.com",
		CAName:      caConfig.CAName,
	}
	enrolmentSecret, err := caClient.Register(adminUser, &registerRequest)
	if err != nil {
		return fmt.Errorf("Error from Register: %s", err)
	}
	fmt.Printf("Registered User: %s, Secret: %s", userName, enrolmentSecret)
	// Enrol the previously registered user
	//ekey, ecert, err := caClient.Enroll(userName, enrolmentSecret)
	_, _, err = caClient.Enroll(userName, enrolmentSecret)
	if err != nil {
		return fmt.Errorf("Error enroling user: %s", err.Error())
	}
    return nil

    //what you see below is from fabric_ca_test.go
    /*
	mspID, err := testFabricConfig.MspID(org1Name)
	if err != nil {
		t.Fatalf("GetMspId() returned error: %v", err)
	}

	caConfig, err := testFabricConfig.CAConfig(org1Name)
	if err != nil {
		t.Fatalf("GetCAConfig returned error: %s", err)
	}

	client := client.NewClient(testFabricConfig)

	cryptoSuiteProvider, err := cryptosuite.GetSuiteByConfig(testFabricConfig)
	if err != nil {
		t.Fatalf("Failed getting cryptosuite from config : %s", err)
	}

	stateStorePath := "/tmp/enroll_user"
	client.SetCryptoSuite(cryptoSuiteProvider)
	stateStore, err := kvs.NewFileKeyValueStore(&kvs.FileKeyValueStoreOptions{
		Path: stateStorePath,
		KeySerializer: func(key interface{}) (string, error) {
			keyString, ok := key.(string)
			if !ok {
				return "", errors.New("converting key to string failed")
			}
			return path.Join(stateStorePath, keyString+".json"), nil
		},
	})
	if err != nil {
		t.Fatalf("CreateNewFileKeyValueStore return error[%s]", err)
	}
	client.SetStateStore(stateStore)

	caClient, err := fabricCAClient.NewFabricCAClient(org1Name, testFabricConfig, cryptoSuiteProvider)
	if err != nil {
		t.Fatalf("NewFabricCAClient return error: %v", err)
	}

	// Admin user is used to register, enroll and revoke a test user
	adminUser, err := client.LoadUserFromStateStore("admin")

	if err != nil {
		t.Fatalf("client.LoadUserFromStateStore return error: %v", err)
	}
	if adminUser == nil {
		key, cert, err := caClient.Enroll("admin", "adminpw")
		if err != nil {
			t.Fatalf("Enroll return error: %v", err)
		}
		if key == nil {
			t.Fatalf("private key return from Enroll is nil")
		}
		if cert == nil {
			t.Fatalf("cert return from Enroll is nil")
		}

		certPem, _ := pem.Decode(cert)
		if certPem == nil {
			t.Fatal("Fail to decode pem block")
		}

		cert509, err := x509.ParseCertificate(certPem.Bytes)
		if err != nil {
			t.Fatalf("x509 ParseCertificate return error: %v", err)
		}
		if cert509.Subject.CommonName != "admin" {
			t.Fatalf("CommonName in x509 cert is not the enrollmentID")
		}
		adminUser2 := identity.NewUser("admin", mspID)
		adminUser2.SetPrivateKey(key)
		adminUser2.SetEnrollmentCertificate(cert)
		err = client.SaveUserToStateStore(adminUser2)
		if err != nil {
			t.Fatalf("client.SaveUserToStateStore return error: %v", err)
		}
		adminUser, err = client.LoadUserFromStateStore("admin")
		if err != nil {
			t.Fatalf("client.LoadUserFromStateStore return error: %v", err)
		}
		if adminUser == nil {
			t.Fatalf("client.LoadUserFromStateStore return nil")
		}
	}

	// Register a random user
	userName := createRandomName()
	registerRequest := ca.RegistrationRequest{
		Name:        userName,
		Type:        "user",
		Affiliation: "org1.department1",
		CAName:      caConfig.CAName,
	}
	enrolmentSecret, err := caClient.Register(adminUser, &registerRequest)
	if err != nil {
		t.Fatalf("Error from Register: %s", err)
	}
	t.Logf("Registered User: %s, Secret: %s", userName, enrolmentSecret)
	// Enrol the previously registered user
	ekey, ecert, err := caClient.Enroll(userName, enrolmentSecret)
	if err != nil {
		t.Fatalf("Error enroling user: %s", err.Error())
	}
    */
}
