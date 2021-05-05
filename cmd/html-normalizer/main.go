package main

import (
	"bytes"
	"container/list"
	"context"
	"io"

	"github.com/gin-gonic/gin"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/gen/pb"
	"golang.org/x/net/html"
	"google.golang.org/grpc"
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
}

func main() {

	var tokenizerOpts []grpc.DialOption
	tokenizerOpts = append(tokenizerOpts, grpc.WithInsecure())
	tokenizerOpts = append(tokenizerOpts, grpc.WithBlock())
	tokenizerConn, err := grpc.Dial(":50051", tokenizerOpts...)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := tokenizerConn.Close(); err != nil {
			panic(err)
		}
	}()
	tokenizerClient := pb.NewTokenizerClient(tokenizerConn)

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

	startHttpServer(tokenizerClient, dictionaryClient)
}

func startHttpServer(tokenizerClient pb.TokenizerClient, dictionaryClient pb.RecognizerClient) {
	r := gin.Default()
	r.POST("/", func(c *gin.Context) {
		tokenizer, err := tokenizerClient.Tokenize(context.Background())
		if err != nil {
			_ = c.AbortWithError(500, err)
			return
		}

		dictionary, err := dictionaryClient.Recognize(context.Background())
		if err != nil {
			_ = c.AbortWithError(500, err)
			return
		}

		tokenizerErrorChan := make(chan error)
		go func() {
			for {
				token, err := tokenizer.Recv()
				if err == io.EOF {
					tokenizerErrorChan <- nil
					return
				} else if err != nil {
					tokenizerErrorChan <- err
					return
				}

				if err := dictionary.Send(token); err != nil {
					tokenizerErrorChan <- err
				}
			}
		}()

		dictionaryErrorChan := make(chan error)
		var entities []*pb.RecognizedEntity
		go func() {
			for {
				entity, err := dictionary.Recv()
				if err == io.EOF {
					dictionaryErrorChan <- nil
					return
				} else if err != nil {
					dictionaryErrorChan <- err
					return
				}
				entities = append(entities, entity)
			}
		}()

		onSnippet := func(b []byte, position uint32) error {
			err := tokenizer.Send(&pb.Snippet{
				Data:   string(b),
				Offset: position,
			})
			return err
		}

		if err := normalize(c.Request.Body, onSnippet); err != nil {
			_ = c.AbortWithError(500, err)
			return
		}

		if err := tokenizer.CloseSend(); err != nil {
			_ = c.AbortWithError(500, err)
			return
		}

		if err = <-tokenizerErrorChan; err != nil {
			_ = c.AbortWithError(500, err)
			return
		}

		if err := dictionary.CloseSend(); err != nil {
			_ = c.AbortWithError(500, err)
			return
		}

		if err = <-dictionaryErrorChan; err != nil {
			_ = c.AbortWithError(500, err)
			return
		}

		c.JSON(200, entities)
	})

	r.POST("/text", func(c *gin.Context) {
		var text []byte
		onSnippet := func(b []byte, _ uint32) error {
			text = append(text, b...)
			return nil
		}
		if err := normalize(c.Request.Body, onSnippet); err != nil {
			_ = c.AbortWithError(500, err)
			return
		}

		c.Data(200, "text/plain", text)
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

func normalize(r io.Reader, onSnippet func([]byte, uint32) error) error {
	tokenizer := html.NewTokenizer(r)
	var position uint32
	var stack Stack
	buf := bytes.NewBuffer([]byte{})

Loop:
	for {
		token := tokenizer.Next()
		switch token {
		case html.ErrorToken:
			break Loop
		case html.TextToken:
			b := tokenizer.Text()
			if _, disallowed := disallowedNodes[stack.Top().name]; !disallowed {
				buf.Write(b)
			}
			position += uint32(len(b))
		case html.StartTagToken:
			b := tokenizer.Raw()
			tn, _ := tokenizer.TagName()
			stack.Push(Tag{name: string(tn), start: position})
			position += uint32(len(html.UnescapeString(string(b))))
		case html.EndTagToken:
			b := tokenizer.Raw()
			tn, _ := tokenizer.TagName()
			if _, disallowed := disallowedNodes[stack.Top().name]; !disallowed && string(tn) == stack.Top().name {
				if _, breakLine := linebreakNodes[stack.Top().name]; breakLine {
					buf.Write([]byte{'\n'})
				}
				bufferBytes, err := buf.ReadBytes(0)
				if err != nil && err != io.EOF {
					return err
				}
				if err = onSnippet(bufferBytes, stack.Top().start); err != nil {
					return err
				}
			}
			position += uint32(len(html.UnescapeString(string(b))))
			stack.Pop()
		case html.SelfClosingTagToken:
			b := tokenizer.Raw()
			tn, _ := tokenizer.TagName()
			if _, breakLine := linebreakNodes[string(tn)]; breakLine {
				buf.Write([]byte{'\n'})
			}
			position += uint32(len(html.UnescapeString(string(b))))
		default:
			b := tokenizer.Raw()
			position += uint32(len(html.UnescapeString(string(b))))
		}
	}
	if err := tokenizer.Err(); err != io.EOF {
		return err
	}
	return nil
}
