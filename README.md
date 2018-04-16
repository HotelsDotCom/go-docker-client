# Go Docker Client
Library to make using docker with Golang easier.

## Installation
* Clone this repo
* Run `dep ensure` (must have [dep](https://github.com/golang/dep) installed )

## To run these tests via console, run:
```
go test ./...
```
To run the integration test too, run:
```
go test ./... -tags=integration
```



## Example

```
import (
       "github.com/HotelsDotCom/go-docker-client"
       "fmt"
)

func main() {
    // Start a new dockerClient
    dockerClient, err := docker.NewDocker()
    if err != nil {
           fmt.Errorf("unable to create client: %s", err)
    }
  
    err = dockerClient.Pull("imagePath")
    if err != nil {
           fmt.Errorf("unable to pull image: %s", err)
    }

    aContainer, err = dockerClient.Run("container-name", "image-path", []string{"BANANA=YELLOW"}, []string{"8080/tcp","80:80","127.0.0.1"})
    if err != nil {
           fmt.Errorf("unable to start container: %s", err)
    }

    ip, err = aContainer.GetIP()
    if err != nil {
           fmt.Errorf("unable to get container IP address: %s", err)
    }

    fmt.Printf("this is the IP address of the container: %s",ip)
    
    err = aContainer.Stop()
    if err != nil {
           fmt.Errorf("unable to stop container: %s", err)
    }
}
```