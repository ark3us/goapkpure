package goapkpure

import (
	"fmt"
	"github.com/Danny-Dasilva/CycleTLS/cycletls"
	"github.com/PuerkitoBio/goquery"
	"github.com/RomainMichau/cloudscraper_go/cloudscraper"
	"io"
	"log"
	"regexp"
	"strings"
)

type VerItem struct {
	Url          string
	Version      string
	VersionCode  int
	DownloadUrl  string
	Downloads    string
	Size         string
	Title        string
	UpdateOn     string
	Signature    string
	Sha1         string
	AndroidVer   string
	Architecture string
	ScreenDPI    string
	BaseApk      string
	SplitApk     string
}

const URL_BASE = "https://apkpure.com"
const URL_DOWNLOAD = "https://d.apkpure.com/b/APK/%s?versionCode=%d"

var logDebug = log.New(io.Discard, "[D] ", log.LstdFlags|log.Lshortfile)
var logInfo = log.New(log.Writer(), "[I] ", log.LstdFlags|log.Lshortfile)
var logError = log.New(log.Writer(), "[E] ", log.LstdFlags|log.Lshortfile)
var logWarn = log.New(log.Writer(), "[W] ", log.LstdFlags|log.Lshortfile)

func EnableDebug(enable bool) {
	if enable {
		logDebug.SetOutput(log.Writer())
	} else {
		logDebug.SetOutput(io.Discard)
	}
}

func httpGet(url string) (string, error) {
	client, err := cloudscraper.Init(false, false)
	if err != nil {
		logError.Println("Failed to initialize cloudscraper client:", err)
		return "", err
	}
	options := cycletls.Options{
		Headers: map[string]string{},
		Timeout: 10,
	}
	res, err := client.Do(url, options, "GET")
	if err != nil {
		logError.Println("Error fetching URL:", url, "->", err)
		return "", err
	}
	if res.Status != 200 {
		logError.Println("Received non-200 status code:", res.Status, "for URL:", url)
		return "", fmt.Errorf("received non-200 status code: %d", res.Status)
	}
	return res.Body, nil
}

func GetPackagePageUrl(packageName string) (string, error) {
	res, err := httpGet(URL_BASE + "/search?q=" + packageName)
	if err != nil {
		logError.Println("Error fetching package page:", err)
		return "", err
	}
	r, err := regexp.Compile("/([^/]*)/" + packageName)
	if err != nil {
		return "", err
	}
	match := r.FindString(res)
	logDebug.Println("Regex match found:", match)
	return URL_BASE + match, nil
}

func parseVersionCode(versionStr string) int {
	r, err := regexp.Compile(`\d+`)
	if err != nil {
		logError.Println("Error compiling regex for version code:", err)
		return 0
	}
	matches := r.FindAllString(versionStr, -1)
	if len(matches) == 0 {
		logError.Println("No version code found in string:", versionStr)
		return 0
	}
	var versionCode int
	fmt.Sscanf(matches[0], "%d", &versionCode)
	return versionCode
}

func (s *VerItem) GetVariants() ([]VerItem, error) {
	if s.Url == "" {
		logError.Println("App URL is empty, cannot fetch version details")
		return nil, fmt.Errorf("app URL is empty, cannot fetch version details")
	}
	res, err := httpGet(s.Url)
	if err != nil {
		logError.Println("Error fetching version details for URL:", s.Url, "->", err)
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(res))
	if err != nil {
		logError.Println("Error parsing version details for URL:", s.Url, "->", err)
		return nil, err
	}
	downloads := ""
	title := doc.Find(".info-title").Text()
	logDebug.Println("Title:", title)
	version := doc.Find(".version-name").Text()
	logDebug.Println("Version:", version)
	headItems := doc.Find(".dev-partnership-head-info li")
	if headItems.Length() == 5 {
		downloads = headItems.Eq(2).Find(".head").Text()
		logDebug.Println("Downloads:", downloads)
	} else {
		logWarn.Println("Unexpected number of head items for version:", s.Url, "->", headItems.Length())
	}

	variantNodes := doc.Find("#version-list .apk")
	if variantNodes.Length() == 0 {
		logError.Println("No variants found in the document for URL:", s.Url)
		return nil, fmt.Errorf("no variants found in the document for URL: %s", s.Url)
	}

	logInfo.Println("Found", variantNodes.Length(), "variant nodes for URL:", s.Url)

	var verItems []VerItem
	for i := 0; i < variantNodes.Length(); i++ {
		verItem := VerItem{}
		variantNode := variantNodes.Eq(i)

		verItem.Url = s.Url
		verItem.Downloads = downloads
		verItem.Title = title
		verItem.Version = version
		verItem.DownloadUrl = variantNode.Find(".download-btn").AttrOr("href", "")
		logDebug.Println("Download URL:", verItem.DownloadUrl)

		verItem.VersionCode = parseVersionCode(variantNode.Find(".code").Text())
		logDebug.Println("Version Code:", verItem.VersionCode)

		verItem.UpdateOn = variantNode.Find("span.time").Text()
		logDebug.Println("Update On:", verItem.UpdateOn)
		verItem.Size = variantNode.Find("span.size").Text()
		logDebug.Println("Size:", verItem.Size)
		verItem.AndroidVer = variantNode.Find("span.sdk").Text()
		logDebug.Println("Android Version:", verItem.AndroidVer)

		infoDetails := map[string]string{}
		infoItems := variantNode.Find(".variants-desc-dialog .content p")
		for i := 0; i < infoItems.Length(); i++ {
			label := strings.TrimSpace(infoItems.Eq(i).Find(".label").Text())
			value := strings.TrimSpace(infoItems.Eq(i).Find(".value").Text())
			infoDetails[label] = value
			logDebug.Println("Info Item:", label, "->", value)
		}

		verItem.Architecture = infoDetails["Architecture"]
		verItem.AndroidVer = infoDetails["Requires Android"]
		verItem.Signature = infoDetails["Signature"]
		verItem.ScreenDPI = infoDetails["Screen DPI"]
		verItem.BaseApk = infoDetails["Base APK"]
		verItem.SplitApk = infoDetails["Split APK"]
		verItem.Sha1 = infoDetails["File SHA1"]

		verItems = append(verItems, verItem)
	}

	logInfo.Println("Successfully fetched", len(verItems), "version details for URL:", s.Url)
	return verItems, nil
}

func GetVersions(packageName string) ([]VerItem, error) {
	packageUrl, err := GetPackagePageUrl(packageName)
	if err != nil {
		logError.Println("Error fetching package page URL:", err)
		return nil, err
	}
	url := packageUrl + "/versions"
	res, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(res))
	if err != nil {
		logError.Println("Error parsing document:", err)
		return nil, err
	}

	title := strings.TrimSpace(doc.Find(".ver_title h1").Text())

	versionNodes := doc.Find(".ver_download_link")
	logDebug.Println("Found", versionNodes.Length(), "version nodes for package:", packageName)

	var verItems []VerItem
	for i := 0; i < versionNodes.Length(); i++ {
		node := versionNodes.Eq(i)
		verItem := VerItem{}
		verItem.Title = title
		verItem.Url = node.AttrOr("href", "")
		verItem.Version = node.AttrOr("data-dt-version", "")
		verItem.Size = node.Find(".ver-item-s").Text()
		verItem.UpdateOn = node.Find(".update-on").Text()
		verItem.VersionCode = parseVersionCode(node.AttrOr("data-dt-versioncode", "0"))
		verItem.DownloadUrl = fmt.Sprintf(URL_DOWNLOAD, packageName, verItem.VersionCode)
		verItems = append(verItems, verItem)
	}
	if len(verItems) == 0 {
		logError.Println("No versions found for package:", packageName)
		return nil, fmt.Errorf("no versions found for package: %s", packageName)
	}
	logInfo.Println("Successfully fetched", len(verItems), "versions for package:", packageName)
	return verItems, nil
}
