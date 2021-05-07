# mcacheZero
Key/Value Store Caching System with LRU Algorithm.

*This project is developed for personal use - You can use it at your own risk and purpose !!!*


**Example Usage: Write Always Mode**

```go
package main

import (
	"fmt"
	"github.com/authapon/mcachezero"
	"time"
	"errors"
)

func main() {
	c := mcachezero.New(2)  //make cache with size 2
	c.SetWriteAlways()  //default Write Mode - call write function every time set data
	c.SetWriteFunc(writer)   // define the write function
	c.SetReadFunc(reader)

	data, err := c.Get("a1")
	if err != nil {
		fmt.Printf("error\n")
	} else {
		fmt.Printf("Get a1 - %s\n", data.(string))
	}

	data, err = c.Get("a2")
	if err != nil {
		fmt.Printf("error\n")
	} else {
		fmt.Printf("Get a2 - %s\n", data.(string))
	}

	data, err = c.Get("a3")
	if err != nil {
		fmt.Printf("Error reading a3\n")
	} else {
		fmt.Printf("Get a3 - %s\n", data.(string))
	}

	fmt.Printf("Set a4 - ddddddd\n")
	c.Set("a4", "ddddddd")

	fmt.Printf("Set a5 - eeeeeee\n")
	c.Set("a5", "eeeeeee")

	time.Sleep(5*time.Hour)
}

func writer(key string, value interface{}) {
	fmt.Printf("Write %s - %s\n", key, value.(string))
}

func reader(key string)(interface{}, error) {
	if key == "a1" {
		return "aaaaaa", nil
	} else if key == "a2" {
		return "bbbbbb", nil
	} else {
		return nil, errors.New("Data not found")
	}
}
```

**Output:**

Get a1 - aaaaaa  
Get a2 - bbbbbb  
Error reading a3  
Set a4 - ddddddd  
Set a5 - eeeeeee  
Write a4 - ddddddd  
Write a5 - eeeeeee  

---
**Example Usage: Write Evict Mode**

```go
package main

import (
	"errors"
	"fmt"
	"github.com/authapon/mcachezero"
	"time"
)

func main() {
	c := mcachezero.New(2) //make cache with size 2
	c.SetWriteEvict()      //Write Evict Mode - call write function when data is evicted
	c.SetWriteFunc(writer) // define the write function
	c.SetReadFunc(reader)
	c.SetDeleteFunc(deleter)

	data, err := c.Get("a1")
	if err != nil {
		fmt.Printf("error\n")
	} else {
		fmt.Printf("Get a1 - %s\n", data.(string))
	}

	data, err = c.Get("a2")
	if err != nil {
		fmt.Printf("error\n")
	} else {
		fmt.Printf("Get a2 - %s\n", data.(string))
	}

	data, err = c.Get("a3")
	if err != nil {
		fmt.Printf("Error reading a3\n")
	} else {
		fmt.Printf("Get a3 - %s\n", data.(string))
	}

	fmt.Printf("Set a4 - ddddddd\n")
	c.Set("a4", "ddddddd")

	fmt.Printf("Set a5 - eeeeeee\n")
	c.Set("a5", "eeeeeee")

	fmt.Printf("Set a1 - zzzzzz\n")
	c.Set("a1", "zzzzzz")

	data, err = c.Get("a1")
	if err != nil {
		fmt.Printf("error\n")
	} else {
		fmt.Printf("Get a1 - %s\n", data.(string))
	}
	fmt.Printf("Sync !!!\n")
	c.Sync()

	fmt.Printf("Try to delete a1\n")
	c.Delete("a1")

	time.Sleep(5 * time.Hour)
}

func writer(key string, value interface{}) {
	fmt.Printf("Write %s - %s\n", key, value.(string))
}

func reader(key string) (interface{}, error) {
	if key == "a1" {
		return "aaaaaa", nil
	} else if key == "a2" {
		return "bbbbbb", nil
	} else {
		return nil, errors.New("Data not found")
	}
}

func deleter(key string) {
	fmt.Printf("Delete %s\n", key)
}
```

**output:**

Get a1 - aaaaaa  
Get a2 - bbbbbb  
Error reading a3  
Set a4 - ddddddd  
Set a5 - eeeeeee  
Set a1 - zzzzzz  
Get a1 - zzzzzz  
Sync !!!  
Write a4 - ddddddd  
Try to delete a1  
Write a1 - zzzzzz  
Write a5 - eeeeeee  
Delete a1  

