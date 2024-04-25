package log

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
)

func TestLogReader_Raw(t *testing.T) {
	ctx := context.Background()
	clk := clock.New()
	blobStore := newTestBlobStore()
	logRegistry, err := logger.NewLogRegistry("")
	assert.Nil(t, err)
	logFactory := logger.MakeLogrusLogFactoryStdOut(logRegistry)

	resourceID := models.NewJobID().ResourceID
	descriptor := models.NewLogDescriptor(models.NewTime(clk.Now()), models.LogDescriptorID{}, resourceID)

	var (
		in  []*models.LogEntry
		out []byte
	)

	entry := models.NewLogEntryLine(
		1,
		models.NewTime(clk.Now()),
		"I am test text",
		1,
		nil)
	in = append(in, entry)
	out = append(out, entry.Derived().(models.PlainTextLogEntry).GetText()+"\n"...)

	entry = models.NewLogEntryLine(
		2,
		models.NewTime(clk.Now()),
		"I am test text",
		2,
		nil)
	in = append(in, entry)
	out = append(out, entry.Derived().(models.PlainTextLogEntry).GetText()+"\n"...)

	inBuf, err := json.Marshal(in)
	assert.Nil(t, err)

	err = blobStore.PutBlob(ctx, fmt.Sprintf(logChunkKeyFullFormat, resourceID, descriptor.ID, 3, 1, "foo"), bytes.NewReader(inBuf))
	assert.Nil(t, err)

	logReader := newReader(ctx, logFactory, blobStore, &query{
		descriptors: []*models.LogDescriptor{descriptor},
		startSeqNo:  nil,
		plaintext:   true,
	})
	actual, err := ioutil.ReadAll(logReader)
	assert.Nil(t, err)
	assert.True(t, bytes.Equal(actual, out))
}

func TestLogReader_OneWindowOneBlob(t *testing.T) {
	ctx := context.Background()
	clk := clock.New()
	blobStore := newTestBlobStore()
	logRegistry, err := logger.NewLogRegistry("")
	assert.Nil(t, err)
	logFactory := logger.MakeLogrusLogFactoryStdOut(logRegistry)

	resourceID := models.NewJobID().ResourceID
	descriptor := models.NewLogDescriptor(models.NewTime(clk.Now()), models.LogDescriptorID{}, resourceID)

	var (
		in  []*models.LogEntry
		out []*models.LogEntry
	)

	entry := models.NewLogEntryLine(
		1,
		models.NewTime(clk.Now()),
		"I am test text",
		1,
		nil)
	in = append(in, entry)
	out = append(out, entry)

	entry = models.NewLogEntryLine(
		2,
		models.NewTime(clk.Now()),
		"I am test text",
		2,
		nil)
	in = append(in, entry)
	out = append(out, entry)

	inBuf, err := json.Marshal(in)
	assert.Nil(t, err)

	err = blobStore.PutBlob(ctx, fmt.Sprintf(logChunkKeyFullFormat, resourceID, descriptor.ID, 3, 1, "foo"), bytes.NewReader(inBuf))
	assert.Nil(t, err)

	logReader := newReader(ctx, logFactory, blobStore, &query{
		descriptors: []*models.LogDescriptor{descriptor},
		startSeqNo:  nil,
	})
	outBuf, err := json.Marshal(out)
	assert.Nil(t, err)
	actual, err := ioutil.ReadAll(logReader)
	assert.Nil(t, err)
	assert.True(t, bytes.Equal(actual, outBuf))
}

func TestLogReader_OneWindowMultipleBlobs(t *testing.T) {
	ctx := context.Background()
	clk := clock.New()
	blobStore := newTestBlobStore()
	logRegistry, err := logger.NewLogRegistry("")
	assert.Nil(t, err)
	logFactory := logger.MakeLogrusLogFactoryStdOut(logRegistry)

	resourceID := models.NewJobID().ResourceID
	descriptor := models.NewLogDescriptor(models.NewTime(clk.Now()), models.LogDescriptorID{}, resourceID)

	var (
		in  []*models.LogEntry
		out []*models.LogEntry
	)

	blobStore.reset()

	entry := models.NewLogEntryLine(
		1,
		models.NewTime(clk.Now()),
		"I am test text",
		1,
		nil)
	in = append(in, entry)
	out = append(out, entry)

	entry = models.NewLogEntryLine(
		2,
		models.NewTime(clk.Now()),
		"I am test text",
		2,
		nil)
	in = append(in, entry)
	out = append(out, entry)

	inBuf, err := json.Marshal(in)
	assert.Nil(t, err)
	err = blobStore.PutBlob(ctx, fmt.Sprintf(logChunkKeyFullFormat, resourceID, descriptor.ID, 3, 1, "foo"), bytes.NewReader(inBuf))
	assert.Nil(t, err)

	in = nil
	in = append(in, entry)

	entry = models.NewLogEntryLine(
		3,
		models.NewTime(clk.Now()),
		"I am test text",
		3,
		nil)
	in = append(in, entry)
	out = append(out, entry)

	inBuf, err = json.Marshal(in)
	assert.Nil(t, err)
	err = blobStore.PutBlob(ctx, fmt.Sprintf(logChunkKeyFullFormat, resourceID, descriptor.ID, 4, 2, "foo"), bytes.NewReader(inBuf))
	assert.Nil(t, err)

	logReader := newReader(ctx, logFactory, blobStore, &query{
		descriptors: []*models.LogDescriptor{descriptor},
		startSeqNo:  nil,
	})

	actual, err := ioutil.ReadAll(logReader)
	assert.Nil(t, err)

	outBuf, err := json.Marshal(out)
	assert.Nil(t, err)
	assert.True(t, bytes.Equal(actual, outBuf))

	// Now three overlapping chunks
	in = nil
	in = append(in, entry)

	entry = models.NewLogEntryLine(
		4,
		models.NewTime(clk.Now()),
		"I am test text",
		4,
		nil)
	in = append(in, entry)
	out = append(out, entry)

	inBuf, err = json.Marshal(in)
	assert.Nil(t, err)
	err = blobStore.PutBlob(ctx, fmt.Sprintf(logChunkKeyFullFormat, resourceID, descriptor.ID, 5, 3, "foo"), bytes.NewReader(inBuf))
	assert.Nil(t, err)

	logReader = newReader(ctx, logFactory, blobStore, &query{
		descriptors: []*models.LogDescriptor{descriptor},
		startSeqNo:  nil,
	})

	actual, err = ioutil.ReadAll(logReader)
	assert.Nil(t, err)
	outBuf, err = json.Marshal(out)
	assert.Nil(t, err)
	assert.True(t, bytes.Equal(actual, outBuf))

	// One window, two chunks with a gap in the middle
	blobStore.reset()
	in = nil
	out = nil

	entry = models.NewLogEntryLine(
		1,
		models.NewTime(clk.Now()),
		"I am test text",
		1,
		nil)
	in = append(in, entry)
	out = append(out, entry)

	entry = models.NewLogEntryLine(
		2,
		models.NewTime(clk.Now()),
		"I am test text",
		2,
		nil)
	in = append(in, entry)
	out = append(out, entry)

	inBuf, err = json.Marshal(in)
	assert.Nil(t, err)
	err = blobStore.PutBlob(ctx, fmt.Sprintf(logChunkKeyFullFormat, resourceID, descriptor.ID, 3, 1, "foo"), bytes.NewReader(inBuf))
	assert.Nil(t, err)

	in = nil

	entry = models.NewLogEntryLine(
		5,
		models.NewTime(clk.Now()),
		"I am test text",
		5,
		nil)
	in = append(in, entry)
	out = append(out, entry)

	inBuf, err = json.Marshal(in)
	assert.Nil(t, err)
	err = blobStore.PutBlob(ctx, fmt.Sprintf(logChunkKeyFullFormat, resourceID, descriptor.ID, 6, 5, "foo"), bytes.NewReader(inBuf))
	assert.Nil(t, err)

	logReader = newReader(ctx, logFactory, blobStore, &query{
		descriptors: []*models.LogDescriptor{descriptor},
		startSeqNo:  nil,
	})

	actual, err = ioutil.ReadAll(logReader)
	assert.Nil(t, err)
	outBuf, err = json.Marshal(out)
	assert.Nil(t, err)
	assert.True(t, bytes.Equal(actual, outBuf))

	// One window, two contiguous chunks
	blobStore.reset()
	in = nil
	out = nil

	entry = models.NewLogEntryLine(
		1,
		models.NewTime(clk.Now()),
		"I am test text",
		1,
		nil)
	in = append(in, entry)
	out = append(out, entry)

	entry = models.NewLogEntryLine(
		2,
		models.NewTime(clk.Now()),
		"I am test text",
		2,
		nil)
	in = append(in, entry)
	out = append(out, entry)

	inBuf, err = json.Marshal(in)
	assert.Nil(t, err)
	err = blobStore.PutBlob(ctx, fmt.Sprintf(logChunkKeyFullFormat, resourceID, descriptor.ID, 3, 1, "foo"), bytes.NewReader(inBuf))
	assert.Nil(t, err)

	in = nil

	entry = models.NewLogEntryLine(
		3,
		models.NewTime(clk.Now()),
		"I am test text",
		3,
		nil)
	in = append(in, entry)
	out = append(out, entry)

	inBuf, err = json.Marshal(in)
	assert.Nil(t, err)
	err = blobStore.PutBlob(ctx, fmt.Sprintf(logChunkKeyFullFormat, resourceID, descriptor.ID, 4, 3, "foo"), bytes.NewReader(inBuf))
	assert.Nil(t, err)

	logReader = newReader(ctx, logFactory, blobStore, &query{
		descriptors: []*models.LogDescriptor{descriptor},
		startSeqNo:  nil,
	})

	actual, err = ioutil.ReadAll(logReader)
	assert.Nil(t, err)
	outBuf, err = json.Marshal(out)
	assert.Nil(t, err)
	assert.True(t, bytes.Equal(actual, outBuf))
}

func TestLogReader(t *testing.T) {
	ctx := context.Background()
	clk := clock.New()
	blobStore := newTestBlobStore()
	logRegistry, err := logger.NewLogRegistry("")
	assert.Nil(t, err)
	logFactory := logger.MakeLogrusLogFactoryStdOut(logRegistry)

	resourceID := models.NewJobID().ResourceID
	descriptor := models.NewLogDescriptor(models.NewTime(clk.Now()), models.LogDescriptorID{}, resourceID)

	var (
		in  []*models.LogEntry
		out []*models.LogEntry
	)

	entry := models.NewLogEntryLine(
		1,
		models.NewTime(clk.Now()),
		"I am test text",
		1,
		nil)
	in = append(in, entry)
	in = append(in, entry)
	out = append(out, entry)

	entry2 := models.NewLogEntryLine(
		2,
		models.NewTime(clk.Now()),
		"I am test text",
		2,
		nil)
	in = append(in, entry2)
	in = append(in, entry)
	out = append(out, entry2)

	inBuf, err := json.Marshal(in)
	assert.Nil(t, err)

	err = blobStore.PutBlob(ctx, fmt.Sprintf(logChunkKeyFullFormat, resourceID, descriptor.ID, 3, 1, "foo"), bytes.NewReader(inBuf))
	assert.Nil(t, err)

	err = blobStore.PutBlob(ctx, fmt.Sprintf(logChunkKeyFullFormat, resourceID, descriptor.ID, 3, 1, "foos"), bytes.NewReader(inBuf))
	assert.Nil(t, err)

	logReader := newReader(ctx, logFactory, blobStore, &query{
		descriptors: []*models.LogDescriptor{descriptor},
		startSeqNo:  nil,
	})

	actual, err := ioutil.ReadAll(logReader)
	assert.Nil(t, err)
	outBuf, err := json.Marshal(out)
	assert.Nil(t, err)
	assert.True(t, bytes.Equal(actual, outBuf))
}
