package filewalk

type Limiter struct {
	ch chan struct{}
}

func NewLimiter(n int) Limiter {
	ch := make(chan struct{}, n)
	for i := 0; i < n; i++ {
		ch <- struct{}{}
	}
	return Limiter{ch}
}

func (l *Limiter) Wait() {
	<-l.ch
}

func (l *Limiter) Done() {
	l.ch <- struct{}{}
}
