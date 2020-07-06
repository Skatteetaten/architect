package docker

import (
	"testing"
)

func TestMounting(t *testing.T) {

	/*

		client := NewRegistryClient("http://localhost:5000", nil)

		manifest, err := client.GetManifest("aurora/wingnut11", "latest")
		if err != nil {
			t.Fatal("Failed to fetch manifest")
		}

		for _, layer := range manifest.Layers {
			ok, err := client.LayerExists("bjorn/test", layer.Digest)
			if err != nil {
				t.Fatalf("Unable to check layer: %v", err)
			}
			if ok {
				fmt.Println("Layer exists")
			} else {
				fmt.Println("Mounting ", layer.Digest)
				err := client.MountLayer("aurora/wingnut11", "bjorn/test", layer.Digest)
				if err != nil {
					t.Fatalf("Unable to mount layer %s. Got err=%v", layer.Digest, err)
				}
			}
		}

	*/

}

func TestManipulateDockerRegistry(t *testing.T) {

	/*
		client := NewRegistryClient("http://localhost:5000", nil)

		manifest, err := client.GetManifest("aurora/wingnut11", "latest")
		if err != nil {
			t.Fatal("Failed to fetch manifest")
		}

		containerConfig, err := client.GetContainerConfig("aurora/wingnut11", manifest.Config.Digest)
		if err != nil {
			t.Fatal("Failed to fetch container config")
		}

		//ADD Cmd and the new layer to the container config
		containerConfig.ContainerConfig = OCIContainerConfig{}
		containerConfig.Config.Cmd = []string{"sh", "hello.sh"}

		digest, err := util.CalculateDigestFromArchive("testdata/app-layer.tar.gz")

		containerConfig.RootFs.DiffIds = append(containerConfig.RootFs.DiffIds, digest)

		containerConfigFile, err := ioutil.TempFile("/tmp", "prefix")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(containerConfigFile.Name())

		data, err := json.Marshal(containerConfig)
		if err != nil {
			t.Fatal("Marshal failed")
		}
		_, err = containerConfigFile.Write(data)
		if err != nil {
			t.Fatal("Unable to write to tmp file ")
		}

		info, err := containerConfigFile.Stat()
		if err != nil {
			t.Fatal("Unable to get file stats")
		}

		digest, err = util.CalculateDigest(containerConfigFile.Name())
		if err != nil {
			t.Fatal("ContainerConfig: Digest calculation failed")
		}

		manifest.Config.Size = int(info.Size())
		manifest.Config.Digest = digest

		digest, err = util.CalculateDigest("testdata/app-layer.tar.gz")
		if err != nil {
			t.Fatal("Unable to calculate layer digest")
		}

		size, err := util.CalculateSize("testdata/app-layer.tar.gz")
		if err != nil {
			t.Fatal("Unable to calculate size")
		}

		//Modify the manifest
		manifest.Layers = append(manifest.Layers, Layer{
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Size:      size,
			Digest:    digest,
		})

		manifestFile, err := ioutil.TempFile("/tmp", "prefix")
		if err != nil {
			t.Fatal(err)
		}

		data, err = json.Marshal(manifest)
		if err != nil {
			t.Fatal("Marshal failed")
		}

		_, err = manifestFile.Write(data)
		if err != nil {
			t.Fatal("Failed to write manifest file")
		}

		err = client.PushLayer("testdata/app-layer.tar.gz", "bjorn/test", digest)
		if err != nil {
			t.Errorf("Failed to push layer %v", err)
		}

		digest, err = util.CalculateDigest(containerConfigFile.Name())
		if err != nil {
			t.Fatal("ContainerConfig: Digest calculation failed")
		}

		err = client.PushLayer(containerConfigFile.Name(), "bjorn/test", digest)
		if err != nil {
			t.Fatal("Failed to push container configuration")
		}

		digest, err = util.CalculateDigest("testdata/app-layer.tar.gz")
		if err != nil {
			t.Fatal("Unable to calculate layer digest")
		}

		err = client.PushManifest(manifestFile.Name(), "bjorn/test", []string{"latest", "tull"})
		if err != nil {
			t.Errorf("Failed to push manifest: %v", err)
		}

	*/
}
