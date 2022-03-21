// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package lint

import (
	"strings"

	"github.com/emicklei/proto"
	"github.com/minish144/prototool-arm64-support/internal/text"
)

var messagesHaveCommentsExceptRequestResponseTypesLinter = NewLinter(
	"MESSAGES_HAVE_COMMENTS_EXCEPT_REQUEST_RESPONSE_TYPES",
	`Verifies that all non-extended messages except for request and response types have a comment of the form "// MessageName ...".`,
	checkMessagesHaveCommentsExceptRequestResponseTypes,
)

func checkMessagesHaveCommentsExceptRequestResponseTypes(add func(*text.Failure), dirPath string, descriptors []*FileDescriptor) error {
	return runVisitor(&messagesHaveCommentsExceptRequestResponseTypesVisitor{baseAddVisitor: newBaseAddVisitor(add)}, descriptors)
}

type messagesHaveCommentsExceptRequestResponseTypesVisitor struct {
	baseAddVisitor
	messageNameToMessage map[string]*proto.Message
	requestResponseTypes map[string]struct{}
	nestedMessageNames   []string
}

func (v *messagesHaveCommentsExceptRequestResponseTypesVisitor) OnStart(*FileDescriptor) error {
	v.messageNameToMessage = nil
	v.requestResponseTypes = nil
	v.nestedMessageNames = nil
	return nil
}

func (v *messagesHaveCommentsExceptRequestResponseTypesVisitor) VisitMessage(message *proto.Message) {
	v.nestedMessageNames = append(v.nestedMessageNames, message.Name)
	for _, child := range message.Elements {
		child.Accept(v)
	}
	v.nestedMessageNames = v.nestedMessageNames[:len(v.nestedMessageNames)-1]

	if v.messageNameToMessage == nil {
		v.messageNameToMessage = make(map[string]*proto.Message)
	}
	if len(v.nestedMessageNames) > 0 {
		v.messageNameToMessage[strings.Join(v.nestedMessageNames, ".")+"."+message.Name] = message
	} else {
		v.messageNameToMessage[message.Name] = message
	}
}

func (v *messagesHaveCommentsExceptRequestResponseTypesVisitor) VisitService(service *proto.Service) {
	for _, child := range service.Elements {
		child.Accept(v)
	}
}

func (v *messagesHaveCommentsExceptRequestResponseTypesVisitor) VisitRPC(rpc *proto.RPC) {
	if v.requestResponseTypes == nil {
		v.requestResponseTypes = make(map[string]struct{})
	}
	v.requestResponseTypes[rpc.RequestType] = struct{}{}
	v.requestResponseTypes[rpc.ReturnsType] = struct{}{}
}

func (v *messagesHaveCommentsExceptRequestResponseTypesVisitor) Finally() error {
	if v.messageNameToMessage == nil {
		v.messageNameToMessage = make(map[string]*proto.Message)
	}
	for messageName, message := range v.messageNameToMessage {
		if !message.IsExtend {
			if _, ok := v.requestResponseTypes[messageName]; !ok {
				if !hasGolangStyleComment(message.Comment, message.Name) {
					v.AddFailuref(message.Position, `Message %q needs a comment of the form "// %s ..."`, message.Name, message.Name)
				}
			}
		}
	}
	return nil
}
