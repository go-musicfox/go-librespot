package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/arcspace/go-arc-sdk/stdlib/log"
	"github.com/arcspace/go-arc-sdk/stdlib/process"
	respot "github.com/arcspace/go-librespot/librespot/api-respot"
	_ "github.com/arcspace/go-librespot/librespot/core" // bootstrapping
	"github.com/arcspace/go-librespot/librespot/core/oauth"
)

func TestDownload(t *testing.T) {
	log.UseStockFormatter(24, true)

	host, _ := process.Start(&process.Task{
		Label: "download-test",
		OnClosed: func() {
			fmt.Println("download-test shutdown complete")
		},
	})

	sess, err := startSession(host)
	if err != nil {
		t.Fatalf("startSession error: %v", err)
	}

	err = assetTests(sess)
	if err != nil {
		t.Fatalf("assetTests error: %v", err)
	}

	gracefulStop, immediateStop := log.AwaitInterrupt()

	go func() {
		<-gracefulStop
		host.Info(2, "<-gracefulStop")
		//sess.Mercury().Close()
		host.Close()
	}()

	go func() {
		<-immediateStop
		host.Info(2, "<-immediateStop")
		host.Close()
	}()

	// Block on host shutdown completion
	<-host.Done()

	log.Flush()
}

func assetTests(sess respot.Session) error {

	funcSearch(sess, "CloudNone")
	trackID := "spotify:track:4byboliQX2Dd8d5VIhROdt"
	funcTrack(sess, trackID)

	//fmt.Println("Loading track for play: ", trackID)

	asset, err := sess.PinTrack(trackID, respot.PinOpts{
		StartInternally: true,
	})
	if err != nil {
		return fmt.Errorf("Error pinning track: %s\n", err)
	}

	assetReader, err := asset.NewAssetReader()
	if err != nil {
		return err
	}

	buffer, err := ioutil.ReadAll(assetReader)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(asset.Label(), buffer, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func startSession(host process.Context) (respot.Session, error) {

	// Read flags from commandline
	//blob := flag.String("blob", "blob.bin", "spotify auth blob")
	devicename := flag.String("devicename", "librespot", "name of device")
	flag.Parse()

	ctx := respot.DefaultSessionCtx(*devicename)
	ctx.Context = host

	sess, err := respot.StartNewSession(ctx)
	if err != nil {
		return nil, err
	}

	{
		login := &ctx.Login
		login.Username = os.Getenv("SPOTIFY_USER_ID")
		login.Password = os.Getenv("SPOTIFY_USER_PW")
		login.AuthToken = ""

		if login.Password == "" && login.AuthToken == "" {
			client_id := os.Getenv("SPOTIFY_CLIENT_ID")
			client_secret := os.Getenv("SPOTIFY_CLIENT_SECRET")

			var err error
			login.AuthToken, err = oauth.LoginOAuth(client_id, client_secret, "http://localhost:5000/callback")
			if err != nil {
				return nil, err
			}
		}

		err = sess.Login()
		if err != nil {
			return nil, err
		}
	}

	return sess, nil

}

func funcTrack(sess respot.Session, trackURI string) {
	fmt.Println("Loading track: ", trackURI)

	trackID, track, err := sess.Mercury().GetTrack(trackURI)
	if err != nil {
		fmt.Println("Error loading track: ", err)
		return
	}

	fmt.Printf("Track: %q (%s)", track.GetName(), trackID)
}

func funcSearch(sess respot.Session, keyword string) {
	resp, err := sess.Search(keyword, 12)

	if err != nil {
		fmt.Println("Failed to search:", err)
		return
	}

	res := resp.Results

	fmt.Println("Search results for ", keyword)
	fmt.Println("=============================")

	if res.Error != nil {
		fmt.Println("Search result error:", res.Error)
	}

	fmt.Printf("Albums: %d (total %d)\n", len(res.Albums.Hits), res.Albums.Total)

	for _, album := range res.Albums.Hits {
		fmt.Printf(" => %s (%s)\n", album.Name, album.Uri)
	}

	fmt.Printf("\nArtists: %d (total %d)\n", len(res.Artists.Hits), res.Artists.Total)

	for _, artist := range res.Artists.Hits {
		fmt.Printf(" => %s (%s)\n", artist.Name, artist.Uri)
	}

	fmt.Printf("\nTracks: %d (total %d)\n", len(res.Tracks.Hits), res.Tracks.Total)

	for _, track := range res.Tracks.Hits {
		fmt.Printf(" => %s (%s)\n", track.Name, track.Uri)
	}
}
