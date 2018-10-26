package main

import (
    "fmt"
    "PlatoonTest/blockchain"
    "PlatoonTest/web"
    "PlatoonTest/web/controllers"
    "os"
)

func main() {
    fSetup := blockchain.FabricSetup{
        ChannelID:  "samtest",
        ChannelConfig:  os.Getenv("GOPATH") + "/src/PlatoonTest/artifacts/samtest.channel.tx",

        ChainCodeID: "samtest-service",
        ChainCodeGoPath: os.Getenv("GOPATH"),
        ChainCodePath: "PlatoonTest/chaincode/",

        OrgAdmin:   "Admin",
        OrgName:    "Org1",
        ConfigFile: "config.yaml",

        UserName: "User1",
    }

    err := fSetup.Initialize()
    if err != nil {
        fmt.Printf("Unable to initialize the fabric SDK: %v\n", err)
        return
    }

    err = fSetup.InstallAndInstantiateCC()
    if err != nil {
        fmt.Printf("Unable to install and instantiate the chaincode: %v\n", err)
        return
    }

    /*
    //attempt to create a new user
    user2, err := fSetup.sdk.NewClient(fabsdk.WithUser("User2")).Channel(fSetup.ChannelID)
    if err != nil {
        return fmt.Errorf("failed to create new user {User2}: %v", err)
    }
    fmt.Println("Created user User2")
    setup2 := setup
    setup2.client = user2
    function := "joinPlatoon"
    platID := "abcd"
    carID := ""
    txID, err := setup2.Invoke(function, platID, carID)
    if err != nil {
        fmt.Printf("Unable to %s with args{%s, %s}: %v\n", function, platID, carID, err)
    } else {
        fmt.Printf("Successfully did %s with args {%s, %s} transaction ID: %s", function, platID, carID, txID)
    }
    */


    app := &controllers.Application {
        Fabric: &fSetup,
    }

    web.Serve(app)
}
/*
    function := "joinPlatoon"
    platID := "abcd"
    carID := "car1"
    txID, err := fSetup.Invoke(function, platID, carID)
    if err != nil {
        fmt.Printf("Unable to %s with args{%s, %s}: %v\n", function, platID, carID, err)
    } else {
        fmt.Printf("Successfully did %s with args {%s, %s} transaction ID: %s", function, platID, carID, txID)
    }

    function = "joinPlatoon"
    platID = "abcd"
    carID = "car2"
    txID, err = fSetup.Invoke(function, platID, carID)
    if err != nil {
        fmt.Printf("Unable to %s with args{%s, %s}: %v\n", function, platID, carID, err)
    } else {
        fmt.Printf("Successfully did %s with args {%s, %s} transaction ID: %s", function, platID, carID, txID)
    }

    function = "joinPlatoon"
    platID = "abcd"
    carID = "car3"
    txID, err = fSetup.Invoke(function, platID, carID)
    if err != nil {
        fmt.Printf("Unable to %s with args{%s, %s}: %v\n", function, platID, carID, err)
    } else {
        fmt.Printf("Successfully did %s with args {%s, %s} transaction ID: %s", function, platID, carID, txID)
    }

    function = "joinPlatoon"
    platID = "abcd"
    carID = "car4"
    txID, err = fSetup.Invoke(function, platID, carID)
    if err != nil {
        fmt.Printf("Unable to %s with args{%s, %s}: %v\n", function, platID, carID, err)
    } else {
        fmt.Printf("Successfully did %s with args {%s, %s} transaction ID: %s", function, platID, carID, txID)
    }

    function = "joinPlatoon"
    platID = "abcd"
    carID = "car5"
    txID, err = fSetup.Invoke(function, platID, carID)
    if err != nil {
        fmt.Printf("Unable to %s with args{%s, %s}: %v\n", function, platID, carID, err)
    } else {
        fmt.Printf("Successfully did %s with args {%s, %s} transaction ID: %s", function, platID, carID, txID)
    }

    response, err := fSetup.QueryVal(platID)
    if err != nil {
        fmt.Printf("unable to query %s on the chaincode: %v\n", platID, err)
    } else {
        fmt.Printf("Response from querying %s: %s\n", platID, response)
    }

    function = "leavePlatoon"
    platID = "abcd"
    carID = "car2"
    txID, err = fSetup.Invoke(function, platID, carID)
    if err != nil {
        fmt.Printf("Unable to %s with args{%s, %s}: %v\n", function, platID, carID, err)
    } else {
        fmt.Printf("Successfully did %s with args {%s, %s} transaction ID: %s", function, platID, carID, txID)
    }

    response, err = fSetup.QueryVal(platID)
    if err != nil {
        fmt.Printf("unable to query %s on the chaincode: %v\n", platID, err)
    } else {
        fmt.Printf("Response from querying %s: %s\n", platID, response)
    }

    */
