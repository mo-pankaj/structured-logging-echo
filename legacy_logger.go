package main

import (
	"errors"
	"fmt"
	"log"
)

func getFruitByIndex(index int, fruits ...string) (string, error) {
	if len(fruits) < index || index < 0 {
		return "", errors.New("not valid index")
	}
	return fruits[index], nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmsgprefix)
	list := []string{"apple", "orange", "banana", "kivi"}
	fruit, err := getFruitByIndex(1, list...)
	if err != nil {
		log.Printf("Error in fetching fruit. Error: %s", err.Error())
	} else {
		fmt.Printf("You will choosed fruit %s\n", fruit)
	}

	fruit, err = getFruitByIndex(5, list...)
	if err != nil {
		log.Printf("Error in fetching fruit. Error: %s", err.Error())
	} else {
		fmt.Printf("You will choosed fruit %s\n", fruit)
	}

}
