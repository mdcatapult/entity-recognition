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

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithBlock())
	conn, err := grpc.Dial("localhost:50051", opts...)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			panic(err)
		}
	}()
	client := pb.NewTokenizerClient(conn)

	startHttpServer(client)
}

func startHttpServer(client pb.TokenizerClient) {
	r := gin.Default()
	r.POST("/", func(c *gin.Context) {
		tokenizer, err := client.Tokenize(context.Background())
		if err != nil {
			_ = c.AbortWithError(500, err)
			return
		}

		done := make(chan struct{})
		var tokens []*pb.Snippet
		go func() {
			for {
				token, err := tokenizer.Recv()
				if err == io.EOF {
					done <- struct{}{}
					break
				} else if err != nil {
					_ = c.AbortWithError(500, err)
					return
				}
				tokens = append(tokens, token)
			}
		}()

		onSnippet := func(b []byte, position uint32) {
			err := tokenizer.Send(&pb.Snippet{
				Data:   string(b),
				Offset: position,
			})
			if err != nil {
				_ = c.AbortWithError(500, err)
				return
			}
		}

		err = normalize(c.Request.Body, onSnippet)
		if err != nil {
			_ = c.AbortWithError(500, err)
			return
		}

		if err := tokenizer.CloseSend(); err != nil {
			_ = c.AbortWithError(500, err)
			return
		}

		<-done
		c.JSON(200, tokens)
	})
	r.POST("/text", func(c *gin.Context) {
		var text []byte
		onSnippet := func(b []byte, _ uint32) {
			text = append(text, b...)
		}
		err := normalize(c.Request.Body, onSnippet)
		if err != nil {
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

func normalize(r io.Reader, onSnippet func([]byte, uint32)) error {
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
				onSnippet(bufferBytes, stack.Top().start)
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
