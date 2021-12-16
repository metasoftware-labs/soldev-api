package modules

import (
	"api/internal/database"
	"api/internal/providers"
	"api/internal/types"
	"errors"
	"os"
)

// YoutubeIntegration find all YouTube playlists in our DB, using the
// YouTube API fetch all the information and then saves the information
// in the DB
func YoutubeIntegration() error {
	// Get YouTube Playlists
	playlists, err := database.GetPlaylistsByProvider("Youtube")
	if err != nil {
		return err
	}

	var content []types.Content
	// Find videos for each playlists
	for _, playlist := range playlists {

		youtubeData, _ := providers.GetYoutubePlaylist(playlist)

		content = append(content, youtubeData...)
	}

	// If there are no content, return an error
	if content == nil {
		return errors.New("404")
	}

	// Save the videos of each playlist in the content table
	for _, item := range content {
		err = database.SaveContent(item)
		if err != nil {
			return err
		}
	}

	return nil
}

func TwitchIntegration() error {
	// Fetch Twitch from streams from the Science and Technology channel
	content, err := providers.FetchSavedStreams(os.Getenv("TWITCH_SOLANA_ID"))
	if err != nil {
		return err
	}

	// If there are no content, return an error
	if content == nil {
		return errors.New("404")
	}

	// Save Twitch streams in the content table
	for _, item := range content {
		err = database.SaveContent(item)
		if err != nil {
			return err
		}
	}

	return nil
}

// TwitchLiveStream processes live streams, right now only Solana channel
func TwitchLiveStream() error {

	content, err := providers.FetchLiveStream(os.Getenv("TWITCH_SOLANA_ID"))
	if err != nil {
		return err
	}

	// The stream did not start yet, we can just exit.
	if content == nil {
		return nil
	}

	// Save Twitch streams in the content table
	for _, item := range content {
		err = database.SaveContent(item)
		if err != nil {
			return err
		}
	}

	return nil
}
