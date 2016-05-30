package dataloader

type Future func() (interface{}, error)

type IndexedValue struct {
	Index int
	Value interface{}
}

func NewFuture(f Future) Future {
	var result interface{}
	var err error

	c := make(chan struct{}, 1)
	go func() {
		defer close(c)
		result, err = f()
	}()

	return func() (interface{}, error) {
		<-c
		return result, err
	}
}

func NewFutureValue(value interface{}) Future {
	return func() (interface{}, error) {
		return value, nil
	}
}

func NewFutureError(err error) Future {
	return func() (interface{}, error) {
		return nil, err
	}
}

func FutureAll(keys []interface{}, fn func(interface{}) (interface{}, error)) ([]interface{}, error) {
	resc, errc := make(chan IndexedValue), make(chan error)

	for i, key := range keys {
		go func(i int, key interface{}) {
			value, err := fn(key)
			if err != nil {
				errc <- err
				return
			}
			resc <- IndexedValue{Index: i, Value: value}
		}(i, key)
	}

	values := make([]interface{}, len(keys))

	for i := 0; i < len(values); i++ {
		select {
		case value := <-resc:
			values[value.Index] = value.Value
		case err := <-errc:
			return nil, err
		}
	}

	return values, nil
}
