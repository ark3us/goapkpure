package main

import (
	"flag"
	"fmt"
	"github.com/ark3us/goapkpure"
	"log"
)

func main() {
	packagePtr := flag.String("package", "", "Package name")
	flag.Parse()
	if *packagePtr == "" {
		log.Fatalln("Please specify package name with -package flag")
	}

	verItems, _ := goapkpure.GetVersions(*packagePtr)
	if len(verItems) == 0 {
		return
	}
	fmt.Printf("Versions for package %s:\n", *packagePtr)
	for i, verItem := range verItems {
		fmt.Printf("%3d | title=%s | version=%-16s | updateOn=%s | size=%-8s \n | url=%s \n | downloadUrl=%s\n",
			i, verItem.Title, verItem.Version, verItem.UpdateOn, verItem.Size, verItem.Url, verItem.DownloadUrl)
	}

	variants, _ := verItems[0].GetVariants()
	if len(variants) == 0 {
		log.Println("No variants found for the latest version")
		return
	}

	fmt.Printf("Variants for latest version %s:\n", verItems[0].Version)
	for i, variant := range variants {
		fmt.Printf("%3d | versionCode=%-16d | arch=%-10s | sdk=%-16s | signature=%s | sha1=%s \n | downloadUrl=%s\n",
			i, variant.VersionCode, variant.Architecture, variant.AndroidVer, variant.Signature, variant.Sha1, variant.DownloadUrl)
	}
}
