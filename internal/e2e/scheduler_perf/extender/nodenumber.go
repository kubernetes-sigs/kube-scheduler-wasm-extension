package extender

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"go.uber.org/zap"
	schedulerapi "k8s.io/kube-scheduler/extender/v1"
)

const (
	versionPath  = "/version"
	priorityPath = "/priorities"
	port         = "8080"
)

var e *echo.Echo

func Start() {
	e = echo.New()
	e.Use(middleware.Recover())
	nodenumberHandler, err := newNodeNumberExtender()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		panic(err)
	}
	e.POST(priorityPath, nodenumberHandler.handler)

	if err := e.Start(":" + port); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
}

func Shutdown() {
	fmt.Println("Shutting down the server...")

	if err := e.Shutdown(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		panic(err)
	}
	fmt.Println("Shutted down the server...")
}

// nodeNumberExtender is an example extender that favors nodes that have the number suffix which is the same as the number suffix of the pod name.
type nodeNumberExtender struct {
	logger *zap.Logger
}

const (
	matchScore    int64 = 10
	nonMatchScore int64 = 0
)

func newNodeNumberExtender() (*nodeNumberExtender, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	return &nodeNumberExtender{logger: logger}, nil
}

func (n *nodeNumberExtender) handler(c echo.Context) error {
	// debug what's in the body
	req := &schedulerapi.ExtenderArgs{}
	if err := c.Bind(req); err != nil {
		n.logger.Error("failed to bind request", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	hostPriorityList, err := n.score(req)
	if err != nil {
		n.logger.Error("failed to score", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, hostPriorityList)
}

func (n *nodeNumberExtender) score(args *schedulerapi.ExtenderArgs) (*schedulerapi.HostPriorityList, error) {
	podNameLastChar := args.Pod.Name[len(args.Pod.Name)-1:]
	podSuffixNumber, err := strconv.Atoi(podNameLastChar)
	if err != nil {
		podSuffixNumber = int(podNameLastChar[0]) % 10
	}

	list := make(schedulerapi.HostPriorityList, len(args.Nodes.Items))
	for i, node := range args.Nodes.Items {
		nodeNameLastChar := node.Name[len(node.Name)-1:]
		nodenum, err := strconv.Atoi(nodeNameLastChar)
		if err != nil {
			nodenum = int(nodeNameLastChar[0]) % 10
		}

		score := nonMatchScore
		if podSuffixNumber == nodenum {
			// if match, node get high score.
			score = matchScore
		}

		list[i] = schedulerapi.HostPriority{
			Host:  node.Name,
			Score: score,
		}
	}

	return &list, nil
}
