package controllers

type platoonUser struct {
    ID          string
    CurrPlat    string
    Reputation  float64
    Money       float64
    LastMove    int64
    EfficiencyClass string
}
type platoon struct {
    ID string
    CurrSpeed int
    //timestamp of last change
    LastMove int64
    //distance (in miles) since the leaer was last payed
    Distance float64
    Members []string
}

type platoonDeep struct {
    ID string
    CurrSpeed int
    LastMove int64
    Distance float64
    Members []platoonUser
}
