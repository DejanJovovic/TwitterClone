package db

import (
	"Tweet/data"
	"github.com/gocql/gocql"
)

type TweetRepo interface {
	GetTweets() (data.Tweets, error)
	CreateTweet(p *data.Tweet) (bool, error)
	LikeTweet(id gocql.UUID, username gocql.UUID) bool
	GetTweetsByUser(id int) (data.Tweets, error)
	GetLikes(id gocql.UUID) (data.Likes, error)
}
