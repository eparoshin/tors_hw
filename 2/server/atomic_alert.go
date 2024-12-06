package main

type Alert struct {
    C chan struct{}
}

func NewAlert() Alert {
    return Alert{C: make(chan struct{}, 1)}
}

func (alert Alert) Signal() {
    select {
    case alert.C <- struct{}{}:
    default:
    }
}
