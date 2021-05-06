package main

import (
	"bytes"
	"container/list"
	"context"
	"github.com/gin-gonic/gin"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/lib"
	"golang.org/x/net/html"
	"google.golang.org/grpc"
	"io"
	"sync"
)

var disallowedNodes = map[string]struct{}{
	"area":     {},
	"audio":    {},
	"body":     {},
	"head":     {},
	"html":     {},
	"link":     {},
	"meta":     {},
	"noscript": {},
	"script":   {},
	"source":   {},
	"style":    {},
	"input":    {},
	"textarea": {},
	"video":    {},
}

var linebreakNodes = map[string]struct{}{
	"ol":    {},
	"ul":    {},
	"li":    {},
	"table": {},
	"tbody": {},
	"tr":    {},
	"th":    {},
	"br":    {},
	"h1":    {},
	"h2":    {},
	"h3":    {},
	"h4":    {},
	"h5":    {},
	"h6":    {},
	"p":     {},
	"section":     {},
	"header":     {},
	"article":     {},
	"aside":     {},
	"summary":     {},
	"figure":     {},
	"figcaption":     {},
	"footer":     {},
	"nav":     {},
}

func main() {

	var dictionaryOpts []grpc.DialOption
	dictionaryOpts = append(dictionaryOpts, grpc.WithInsecure())
	dictionaryOpts = append(dictionaryOpts, grpc.WithBlock())
	dictionaryConn, err := grpc.Dial(":50052", dictionaryOpts...)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := dictionaryConn.Close(); err != nil {
			panic(err)
		}
	}()
	dictionaryClient := pb.NewRecognizerClient(dictionaryConn)

	var regexOpts []grpc.DialOption
	regexOpts = append(regexOpts, grpc.WithInsecure())
	regexOpts = append(regexOpts, grpc.WithBlock())
	regexConn, err := grpc.Dial(":50053", regexOpts...)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := dictionaryConn.Close(); err != nil {
			panic(err)
		}
	}()
	regexClient := pb.NewRecognizerClient(regexConn)

	startHttpServer(dictionaryClient, regexClient)
}

func startHttpServer(clients ...pb.RecognizerClient) {
	r := gin.Default()
	r.POST("/html/tokens", func(c *gin.Context) {
		type token struct{
			Token string `json:"token"`
			Offset uint32 `json:"offset"`
		}
		var tokens []token
		onSnippet := func(snippet *pb.Snippet) error {
			return lib.Tokenize(snippet, func(snippet *pb.Snippet) error {
				tokens = append(tokens, token{
					Token:  string(snippet.GetData()),
					Offset: snippet.GetOffset(),
				})
				return nil
			})
		}

		if err := HtmlToText(c.Request.Body, onSnippet); err != nil {
			_ = c.AbortWithError(500, err)
			return
		}

		c.JSON(200, tokens)
	})

	r.POST("/html/entities", func(c *gin.Context) {

		var err error
		errChan := make(chan error, len(clients))
		recognisers := make([]pb.Recognizer_RecognizeClient, len(clients))
		for i, client := range clients {
			recognisers[i], err = client.Recognize(context.Background())
		}


		var entities []*pb.RecognizedEntity
		var mut sync.Mutex
		for _, recogniser := range recognisers {
			go func(recogniser pb.Recognizer_RecognizeClient) {
				for {
					entity, err := recogniser.Recv()
					if err == io.EOF {
						errChan <- nil
						return
					} else if err != nil {
						errChan <- err
						return
					}
					mut.Lock()
					entities = append(entities, entity)
					mut.Unlock()
				}
			}(recogniser)
		}

		onSnippet := func(snippet *pb.Snippet) error {
			return lib.Tokenize(snippet, func(snippet *pb.Snippet) error {
				for _, recogniser := range recognisers {
					if err := recogniser.Send(snippet); err != nil {
						return err
					}
				}
				return nil
			})
		}

		if err := HtmlToText(c.Request.Body, onSnippet); err != nil {
			_ = c.AbortWithError(500, err)
			return
		}

		for _, recogniser := range recognisers {
			if err := recogniser.CloseSend(); err != nil {
				_ = c.AbortWithError(500, err)
				return
			}
		}

		for i := 0; i < len(recognisers); i++ {
			if err = <-errChan; err != nil {
				_ = c.AbortWithError(500, err)
				return
			}
		}

		c.JSON(200, entities)
	})

	r.POST("/html/text", func(c *gin.Context) {
		var data []byte
		onSnippet := func(snippet *pb.Snippet) error {
			data = append(data, snippet.GetData()...)
			return nil
		}
		if err := HtmlToText(c.Request.Body, onSnippet); err != nil {
			_ = c.AbortWithError(500, err)
			return
		}

		c.Data(200, "text/plain", data)
	})
	_ = r.Run(":8083")
}

type Stack struct {
	*list.List
}

type Tag struct {
	name  string
	start uint32
}

func (s *Stack) Push(tag Tag) {
	if s.List == nil {
		s.List = list.New()
	}
	s.PushFront(tag)
}

func (s *Stack) Top() Tag {
	if s.List == nil {
		s.List = list.New()
	}
	e := s.Front()
	if e == nil {
		return Tag{}
	}
	return e.Value.(Tag)
}

func (s *Stack) Pop() {
	e := s.Front()
	s.Remove(e)
}

func HtmlToText(r io.Reader, onSnippet func(snippet *pb.Snippet) error) error {
	htmlTokenizer := html.NewTokenizer(r)
	var position uint32
	var stack Stack
	buf := bytes.NewBuffer([]byte{})

Loop:
	for {
		htmlToken := htmlTokenizer.Next()
		switch htmlToken {
		case html.ErrorToken:
			break Loop
		case html.TextToken:
			htmlTokenBytes := htmlTokenizer.Text()
			if _, disallowed := disallowedNodes[stack.Top().name]; !disallowed {
				buf.Write(htmlTokenBytes)
			}
			position += uint32(len(htmlTokenBytes))
		case html.StartTagToken:
			htmlTokenBytes := htmlTokenizer.Raw()
			tn, _ := htmlTokenizer.TagName()
			stack.Push(Tag{name: string(tn), start: position})
			position += uint32(len(html.UnescapeString(string(htmlTokenBytes))))
		case html.EndTagToken:
			htmlTokenBytes := htmlTokenizer.Raw()
			tn, _ := htmlTokenizer.TagName()
			if _, disallowed := disallowedNodes[stack.Top().name]; !disallowed && string(tn) == stack.Top().name {
				if _, breakLine := linebreakNodes[stack.Top().name]; breakLine {
					buf.Write([]byte{'\n'})
				}
				bufferBytes, err := buf.ReadBytes(0)
				if err != nil && err != io.EOF {
					return err
				}
				if err = onSnippet(&pb.Snippet{
					Data:   bufferBytes,
					Offset: stack.Top().start,
				}); err != nil {
					return err
				}
			}
			position += uint32(len(html.UnescapeString(string(htmlTokenBytes))))
			stack.Pop()
		case html.SelfClosingTagToken:
			htmlTokenBytes := htmlTokenizer.Raw()
			tn, _ := htmlTokenizer.TagName()
			if _, breakLine := linebreakNodes[string(tn)]; breakLine {
				buf.Write([]byte{'\n'})
			}
			position += uint32(len(html.UnescapeString(string(htmlTokenBytes))))
		default:
			htmlTokenBytes := htmlTokenizer.Raw()
			position += uint32(len(html.UnescapeString(string(htmlTokenBytes))))
		}
	}
	if err := htmlTokenizer.Err(); err != io.EOF {
		return err
	}
	return nil
}
