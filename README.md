# Dataloader

> A shoddy port of facebook/dataloader to Go.

## Example Usage

``` go
package main

import (
	"fmt"
	"github.com/stayradiated/dataloader"
	"gopkg.in/redis.v3"
)

func main() {
	// create a redis client
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	dl := dataloader.NewDataLoader(func(data []interface{}) ([]interface{}, error) {
		keys := make([]string, len(data))
		for k, v := range data {
			keys[k] = v.(string)
		}

		fmt.Println(keys)

		return client.MGet(keys...).Result()
	}, nil)

	values, err := dl.LoadMany([]interface{}{
		"myvalue:1",
		"myvalue:2",
	})
	fmt.Println(values, err)
}
```
