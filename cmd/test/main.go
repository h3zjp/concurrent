package main

import (
	"context"
	"fmt"
	"github.com/totegamma/concurrent/client"
)

func main() {
	fmt.Println("Hello, World!")

	client := client.NewClient()
	client.RegisterHostRemap("con1.kokopi.me", "localhost:8080", false)
	entity, err := client.GetEntity(context.Background(), "con1.kokopi.me", "con1edzczj6vcxd3kra8e37ysv8s7u6rlul5ellr9t", nil)
	if err != nil {
		panic(err)
	}

	fmt.Println(entity)

}
