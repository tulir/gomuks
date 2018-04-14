// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2018 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package messages

// TagWithMeta is an open HTML tag with some metadata (e.g. list index, a href value).
type TagWithMeta struct {
	Tag     string
	Counter int
	Meta    string
	Text    string
}

// BlankTag is a blank TagWithMeta object.
var BlankTag = &TagWithMeta{}

// TagArray is a reversed queue for remembering what HTML tags are open.
type TagArray []*TagWithMeta

// Pushb converts the given byte array into a string and calls Push().
func (ta *TagArray) Pushb(tag []byte) {
	ta.Push(string(tag))
}

// Popb converts the given byte array into a string and calls Pop().
func (ta *TagArray) Popb(tag []byte) *TagWithMeta {
	return ta.Pop(string(tag))
}

// Indexb converts the given byte array into a string and calls Index().
func (ta *TagArray) Indexb(tag []byte) {
	ta.Index(string(tag))
}

// IndexAfterb converts the given byte array into a string and calls IndexAfter().
func (ta *TagArray) IndexAfterb(tag []byte, after int) {
	ta.IndexAfter(string(tag), after)
}

// Push adds the given tag to the array.
func (ta *TagArray) Push(tag string) {
	ta.PushMeta(&TagWithMeta{Tag: tag})
}

// Push adds the given tag to the array.
func (ta *TagArray) PushMeta(tag *TagWithMeta) {
	*ta = append(*ta, BlankTag)
	copy((*ta)[1:], *ta)
	(*ta)[0] = tag
}

// Pop removes the given tag from the array.
func (ta *TagArray) Pop(tag string) (removed *TagWithMeta) {
	if (*ta)[0].Tag == tag {
		// This is the default case and is lighter than append(), so we handle it separately.
		removed = (*ta)[0]
		*ta = (*ta)[1:]
	} else if index := ta.Index(tag); index != -1 {
		removed = (*ta)[index]
		*ta = append((*ta)[:index], (*ta)[index+1:]...)
	}
	return
}

// Index returns the first index where the given tag is, or -1 if it's not in the list.
func (ta *TagArray) Index(tag string) int {
	return ta.IndexAfter(tag, -1)
}

// IndexAfter returns the first index after the given index where the given tag is,
// or -1 if the given tag is not on the list after the given index.
func (ta *TagArray) IndexAfter(tag string, after int) int {
	for i := after + 1; i < len(*ta); i++ {
		if (*ta)[i].Tag == tag {
			return i
		}
	}
	return -1
}

// Get returns the first occurrence of the given tag, or nil if it's not in the list.
func (ta *TagArray) Get(tag string) *TagWithMeta {
	return ta.GetAfter(tag, -1)
}

// IndexAfter returns the first occurrence of the given tag, or nil if the given
// tag is not on the list after the given index.
func (ta *TagArray) GetAfter(tag string, after int) *TagWithMeta {
	for i := after + 1; i < len(*ta); i++ {
		if (*ta)[i].Tag == tag {
			return (*ta)[i]
		}
	}
	return nil
}

// Has returns whether or not the list has at least one of the given tags.
func (ta *TagArray) Has(tags ...string) bool {
	for _, tag := range tags {
		if index := ta.Index(tag); index != -1 {
			return true
		}
	}
	return false
}
