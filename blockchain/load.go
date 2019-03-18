package blockchain

import (
    "fmt"
    "time"
    "os"
)
type FuncDef struct {
    Function string
    Arg1 string
    Arg2 string
}

func (setup *ClientSetup) LoadTest(functions []FuncDef, iters int) (string, error) {
    start := time.Now()
    f, err := os.Create(fmt.Sprintf("logs/%s-%d-[%s].log", setup.UserName, iters, start.Format("2006-01-02 15:04:05")))
    if err != nil {
        fmt.Printf("Couldn't open file: %v\n", err)
    }
    defer f.Close()
    _, err = f.WriteString(fmt.Sprintf("%s: [%s] start run\n", start.Format("2006-01-02 15:04:05.000000"), setup.UserName))
    for i := 0; i < iters; i++ {
        for j, def := range(functions) {
            beginTrans := time.Now()
            _, err := f.WriteString(fmt.Sprintf("%s: [%s] start transaction %d:%d\n", beginTrans.Format("2006-01-02 15:04:05.000000"), setup.UserName, i, j))
            if err != nil {
                fmt.Printf("Couldn't write to file: %v\n", err)
            }
            fmt.Printf("Doing %s:%d:%d\n", setup.UserName, i, j)
            _, err = setup.Invoke(def.Function, def.Arg1, def.Arg2)
            for err != nil {
                time.Sleep(100 * time.Millisecond)
                _, err = setup.Invoke(def.Function, def.Arg1, def.Arg2)
            }
            endTrans := time.Now()
            _, err = f.WriteString(fmt.Sprintf("%s: [%s] end transaction %d:%d\n", endTrans.Format("2006-01-02 15:04:05.000000"), setup.UserName, i, j))
        }
    }
    end := time.Now()
    timeFinish := time.Since(start)
    _, err = f.WriteString(fmt.Sprintf("%s: [%s] end run in %d\n", end.Format("2006-01-02 15:04:05.000000"), setup.UserName, timeFinish))
    f.Sync()

    return "all good", nil
}
