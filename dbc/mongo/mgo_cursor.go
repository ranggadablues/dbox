package mongo

import (
	"errors"
	"fmt"
	"github.com/eaciit/dbox"
	"github.com/eaciit/errorlib"
	_ "github.com/eaciit/toolkit"
	"gopkg.in/mgo.v2"
	//"reflect"
)

const (
	modCursor = "Cursor"

	QueryResultCursor = "MongoCursor"
	QueryResultPipe   = "MongoPipe"
)

type Cursor struct {
	dbox.Cursor

	ResultType string
	mgoCursor  *mgo.Query
	mgoPipe    *mgo.Pipe
	mgoIter    *mgo.Iter

	count int

	session          *mgo.Session
	isPoolingSession bool
}

func (c *Cursor) Close() {
	if c.mgoIter != nil {
		c.mgoIter.Close()
	}

	if c.session != nil && !c.isPoolingSession {
		c.session.Close()
	}
}

func (c *Cursor) validate() error {
	if c.ResultType == QueryResultPipe {
		if c.mgoPipe == nil {
			return errors.New(fmt.Sprintf("Pipe is nil"))
		}
	} else if c.ResultType == QueryResultCursor {
		if c.mgoCursor == nil {
			return errors.New("Query cursor is nil")
		}
	}

	return nil
}

func (c *Cursor) prepIter() error {
	e := c.validate()
	if e != nil {
		return e
	}
	if c.mgoIter == nil {
		if c.ResultType == QueryResultPipe {
			c.mgoIter = c.mgoPipe.Iter()
		} else if c.ResultType == QueryResultCursor {
			c.mgoIter = c.mgoCursor.Iter()
		}
	}
	return nil
}

func (c *Cursor) Count() int {
	return c.count
}

func (c *Cursor) ResetFetch() error {
	c.mgoIter = nil
	e := c.prepIter()
	if e != nil {
		return errorlib.Error(packageName, modCursor, "ResetFetch", e.Error())
	}
	return nil
}

func (c *Cursor) Fetch(m interface{}, n int, closeWhenDone bool) (
	*dbox.DataSet, error) {
	if closeWhenDone {
		defer c.Close()
	}

	e := c.prepIter()
	if e != nil {
		return nil, errorlib.Error(packageName, modCursor, "Fetch", e.Error())
	}

	if c.mgoIter == nil {
		return nil, errorlib.Error(packageName, modCursor, "Fetch", "Iter object is not yet initialized")
	}

	ds := dbox.NewDataSet(m)
	if n == 0 {
		datas := []interface{}{}
		e = c.mgoIter.All(&datas)
		if e != nil {
			return ds, errorlib.Error(packageName, modCursor,
				"Fetch", e.Error())
		}
		ds.Data = datas
	} else if n > 0 {
		fetched := 0
		fetching := true
		for fetching {
			dataHolder := m

			if bOk := c.mgoIter.Next(&dataHolder); bOk {
				ds.Data = append(ds.Data, dataHolder)
				fetched++
				if fetched == n {
					fetching = false
				}
			} else {
				fetching = false
			}
		}
	}

	return ds, nil
}