package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

type goHandler func() error

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	// 信号监听
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh)
	go func() {
		for {
			select {
			case s := <-sigCh:
				log.Printf("got signal: %v\n", s)
				if s != os.Interrupt {
					continue
				}
				cancel()
				return
			default:
				time.Sleep(time.Millisecond)
			}
		}
	}()

	r := gin.Default()

	// 默认监听响应
	r.GET("/server", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "server",
		})
	})

	// 服务退出调用
	r.GET("/stop", func(c *gin.Context) {
		cancel()
		c.JSON(http.StatusOK, gin.H{
			"message": "shutting down",
		})
	})

	eg, ctxNew := errgroup.WithContext(ctx)
	eg.Go(runHttpServer(ctxNew, "127.0.0.1:8080", r))
	eg.Go(runHttpServer(ctxNew, "127.0.0.1:9090", r))

	err := eg.Wait()
	if err != nil {
		log.Printf("final err: %+v\n", err)
	}

}

func runHttpServer(ctx context.Context, addr string, g *gin.Engine) goHandler {
	return func() error {
		// 退出打印
		defer log.Printf("listen %s return\n", addr)
		listener := http.Server{
			Addr:    addr,
			Handler: g,
		}
		go func() {
			for {
				select {
				case <-ctx.Done():
					err := listener.Shutdown(ctx)
					if err != nil {
						log.Printf("runHttpServer : %+v\n", err)
					}
					return
				default:
					time.Sleep(time.Second)
				}
			}
		}()
		return listener.ListenAndServe()
	}
}
