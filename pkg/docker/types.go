package docker

import "fmt"

type ImageName struct {
	Repository 	string
	Registry 	string
	Tag		string
}

func (n ImageName) String() string {
	if n.Registry == "" {
		return fmt.Sprintf("%s:%s", n.Repository, n.Tag)
	}

	return fmt.Sprintf("%s/%s:%s", n.Registry, n.Repository, n.Tag)
}
