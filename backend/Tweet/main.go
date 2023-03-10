package main

import (
	"Tweet/db"
	"Tweet/handlers"
	"context"
	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	//Initialize the logger we are going to use, with prefix and datetime for every log
	//As per 12 factor app the general place for app to log is the standard output.
	//If you want to save the logs to a file run the app with the following command.
	//
	//	go run . >> output.txt
	//
	logger := log.New(os.Stdout, "[tweet-api] ", log.LstdFlags)

	timeoutContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	//Load .env file, generally used just to not fill up the environment with variables that are for a specific project
	//Important notice is that .env file is not stored in version control (Git, Svn...) and every developer should create his .env file
	//And fill it with variables for his system. After loading .env variables we can access them with os.Getenv()
	//e := godotenv.Load()
	//if e != nil {
	//	logger.Fatal(e)
	//}

	//Reading from environment, if not set we will default it to 8080.
	//This allows flexibility in different environments (for eg. when running multiple docker api's and want to override the default port)
	port := os.Getenv("app_port")
	if len(port) == 0 {
		port = "8081"
	}

	// NoSQL: Initialize Product Repository store
	tweetRepo, err := db.NewTweetRepoDB(logger)
	if err != nil {
		logger.Fatal(err)
	}
	defer tweetRepo.CloseSession()

	tweetRepo.CreateTables()

	//Initialize the handler and inject said logger
	tweetsHandler := handlers.NewTweetsHandler(logger, tweetRepo)

	//Initialize the router and add a middleware for all the requests
	routerTweet := mux.NewRouter()
	routerTweet.Use(tweetsHandler.MiddlewareContentTypeSet)

	getTweetsRouter := routerTweet.Methods(http.MethodGet).Subrouter()
	getTweetsRouter.HandleFunc("/tweets", tweetsHandler.GetTweets)

	postTweetRouter := routerTweet.Methods(http.MethodPost).Subrouter()
	postTweetRouter.HandleFunc("/tweets", tweetsHandler.CreateTweet)
	postTweetRouter.Use(tweetsHandler.MiddlewareTweetsValidation)

	likeTweetRouter := routerTweet.Methods(http.MethodPost).Subrouter()
	likeTweetRouter.HandleFunc("/tweets/likes", tweetsHandler.LikeTweet)
	likeTweetRouter.Use(tweetsHandler.MiddlewareLikeValidation)

	getLikeTweetRouter := routerTweet.Methods(http.MethodGet).Subrouter()
	getLikeTweetRouter.HandleFunc("/tweets/likes/{id}", tweetsHandler.GetLikes)

	getLikeUserRouter := routerTweet.Methods(http.MethodGet).Subrouter()
	getLikeUserRouter.HandleFunc("/tweets/likes/users/{id}", tweetsHandler.GetLikesUsers)

	getTweetsByUserRouter := routerTweet.Methods(http.MethodGet).Subrouter()
	getTweetsByUserRouter.HandleFunc("/tweets/user/{username}", tweetsHandler.GetTweetsByUser)

	//Set cors. Generally you wouldn't like to set cors to a "*". It is a wildcard and it will match any source.
	//Normally you would set this to a set of ip's you want this api to serve. If you have an associated frontend app
	//you would put the ip of the server where the frontend is running. The only time you don't need cors is when you
	//calling the api from the same ip, or when you are using the proxy (for eg. Nginx)
	cors := gorillaHandlers.CORS(gorillaHandlers.AllowedOrigins([]string{"https://localhost:4200/"}))

	//Initialize the server
	server := http.Server{
		Addr:         ":" + port,         // Addr optionally specifies the TCP address for the server to listen on, in the form "host:port". If empty, ":http" (port 80) is used.
		Handler:      cors(routerTweet),  // handler to invoke, http.DefaultServeMux if nil
		IdleTimeout:  1200 * time.Second, // IdleTimeout is the maximum amount of time to wait for the next request when keep-alives are enabled.
		ReadTimeout:  1 * time.Second,    // ReadTimeout is the maximum duration for reading the entire request, including the body. A zero or negative value means there will be no timeout.
		WriteTimeout: 1 * time.Second,    // WriteTimeout is the maximum duration before timing out writes of the response.
	}

	logger.Println("Server listening on port", port)
	//Distribute all the connections to goroutines
	//go func() {
	//	err := server.ListenAndServe()
	//	if err != nil {
	//		logger.Fatal(err)
	//	}
	//}()

	//HTTPs:
	go func() {
		err := server.ListenAndServeTLS("certificates/group3.crt", "certificates/group3.key")
		if err != nil {
			logger.Fatal(err)
		}
	}()

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGINT)
	signal.Notify(sigCh, syscall.SIGKILL)

	//When we receive an interrupt or kill, if we don't have any current connections the code will terminate.
	//But if we do the code will stop receiving any new connections and wait for maximum of 30 seconds to finish all current requests.
	//After that the code will terminate.
	sig := <-sigCh
	logger.Println("Received terminate, graceful shutdown", sig)
	timeoutContext, _ = context.WithTimeout(context.Background(), 30*time.Second)

	//Try to shut down gracefully
	if server.Shutdown(timeoutContext) != nil {
		logger.Fatal("Cannot gracefully shutdown...")
	}
}
