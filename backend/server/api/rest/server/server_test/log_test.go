package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/client"
	"github.com/buildbeaver/buildbeaver/server/api/rest/client/clienttest"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
)

func TestLogAPI(t *testing.T) {
	ctx := context.Background()

	// Create a test server
	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanup()
	app.RunnerAPIServer.Start()
	defer app.RunnerAPIServer.Stop(ctx)

	// Create a test API client to talk to the server via client certificate authentication
	apiClient, clientCert := clienttest.MakeClientCertificateAPIClient(t, app)

	testCompany := server_test.CreateCompanyLegalEntity(t, ctx, app, "", "", "")

	// Create a runner in order to register the client certificate as a credential
	_ = server_test.CreateRunner(t, ctx, app, "", testCompany.ID, clientCert)

	// Create a repo, commit and build to use logs with
	repo := server_test.CreateRepo(t, ctx, app, testCompany.ID)
	build := server_test.CreateAndQueueBuild(t, ctx, app, repo.ID, testCompany.ID, "")

	t.Run("Single", testSingleLog(app, apiClient, build.ID))
	t.Run("Merged", testMergedLogs(app, apiClient, build.ID))
}

func testSingleLog(app *server_test.TestServer, client *client.APIClient, buildID models.BuildID) func(t *testing.T) {
	return func(t *testing.T) {
		var (
			ctx       = context.Background()
			read      []*models.LogEntry
			blockName = models.ResourceName("stuff")
		)

		entries := []*models.LogEntry{
			models.NewLogEntryLine(1, models.NewTime(time.Now()), "hello", 1, nil),
			models.NewLogEntryLine(2, models.NewTime(time.Now()), "world", 2, nil),
			models.NewLogEntryBlock(3, models.NewTime(time.Now()), "Doing stuff", "stuff", nil),
			models.NewLogEntryLine(4, models.NewTime(time.Now()), "hello", 3, &blockName),
			models.NewLogEntryLine(5, models.NewTime(time.Now()), "hello", 4, nil),
		}
		writeData, err := json.Marshal(entries)
		require.Nil(t, err)

		logDescriptor, err := app.LogService.Create(ctx, nil, models.NewLogDescriptor(
			models.NewTime(time.Now()),
			models.LogDescriptorID{},
			buildID.ResourceID))
		require.Nil(t, err)

		writer, err := client.OpenLogWriteStream(ctx, logDescriptor.ID)
		require.Nil(t, err)
		_, err = io.Copy(writer, &basicReader{r: bytes.NewReader(writeData)})
		require.Nil(t, err)
		writer.Close()

		// Test reading end to end
		reader, err := client.OpenLogReadStream(ctx, logDescriptor.ID, &documents.LogSearchRequest{LogSearch: &models.LogSearch{}})
		require.Nil(t, err)
		readData, err := ioutil.ReadAll(reader)
		require.Nil(t, err)
		reader.Close()
		read = nil
		err = json.Unmarshal(readData, &read)
		require.Nil(t, err)
		require.True(t, structuredLogsEqual(entries, read))

		// Test start offset
		startSeqNo := 3
		reader, err = client.OpenLogReadStream(ctx, logDescriptor.ID, &documents.LogSearchRequest{LogSearch: &models.LogSearch{StartSeqNo: &startSeqNo}})
		require.Nil(t, err)
		readData, err = ioutil.ReadAll(reader)
		require.Nil(t, err)
		reader.Close()
		read = nil
		err = json.Unmarshal(readData, &read)
		require.Nil(t, err)
		require.True(t, structuredLogsEqual(entries[2:], read))

		// Test reading raw
		plaintext := true
		reader, err = client.OpenLogReadStream(ctx, logDescriptor.ID, &documents.LogSearchRequest{LogSearch: &models.LogSearch{Plaintext: &plaintext}})
		require.Nil(t, err)
		readData, err = ioutil.ReadAll(reader)
		require.Nil(t, err)
		reader.Close()
		require.True(t, rawLogsEqual(entries, readData))

		// Read back after sealing - we should now see the end entry
		err = app.LogService.Seal(ctx, nil, logDescriptor.ID)
		require.Nil(t, err)
		reader, err = client.OpenLogReadStream(ctx, logDescriptor.ID, &documents.LogSearchRequest{LogSearch: &models.LogSearch{}})
		require.Nil(t, err)
		readData, err = ioutil.ReadAll(reader)
		require.Nil(t, err)
		reader.Close()
		read = nil
		err = json.Unmarshal(readData, &read)
		require.Nil(t, err)
		require.Equal(t, read[len(read)-1].Kind, models.LogEntryKindEnd)
		require.True(t, structuredLogsEqual(entries, read[:len(read)-1]))
	}
}

func testMergedLogs(app *server_test.TestServer, client *client.APIClient, buildID models.BuildID) func(t *testing.T) {
	return func(t *testing.T) {

		type result struct {
			entries []*entryWithDescriptorID
			err     error
		}

		var (
			ctx         = context.Background()
			resultC     = make(chan *result)
			descriptors []*models.LogDescriptor
		)

		logDescriptor1, err := app.LogService.Create(ctx, nil, models.NewLogDescriptor(
			models.NewTime(time.Now()),
			models.LogDescriptorID{},
			buildID.ResourceID))
		require.Nil(t, err)
		descriptors = append(descriptors, logDescriptor1)

		logDescriptor2, err := app.LogService.Create(ctx, nil, models.NewLogDescriptor(
			models.NewTime(time.Now()),
			logDescriptor1.ID,
			buildID.ResourceID))
		require.Nil(t, err)
		descriptors = append(descriptors, logDescriptor2)

		logDescriptor3, err := app.LogService.Create(ctx, nil, models.NewLogDescriptor(
			models.NewTime(time.Now()),
			logDescriptor2.ID,
			buildID.ResourceID))
		require.Nil(t, err)
		descriptors = append(descriptors, logDescriptor3)

		logDescriptor4, err := app.LogService.Create(ctx, nil, models.NewLogDescriptor(
			models.NewTime(time.Now()),
			logDescriptor2.ID,
			buildID.ResourceID))
		require.Nil(t, err)
		descriptors = append(descriptors, logDescriptor4)

		for _, descriptor := range descriptors {
			go func(descriptor *models.LogDescriptor) {
				entries, err := fillLogDescriptor(ctx, client, descriptor)
				resultC <- &result{
					entries: entries,
					err:     err,
				}
			}(descriptor)
		}

		// Apply the same deterministic sort logic we expect the server to apply
		var results []*entryWithDescriptorID
		for i := 0; i < len(descriptors); i++ {
			res := <-resultC
			require.Nil(t, res.err)
			results = append(results, res.entries...)
		}
		sort.Slice(results, func(i, j int) bool {
			a := results[i]
			aEntry := a.Derived().(models.PersistentLogEntry)
			b := results[j]
			bEntry := b.Derived().(models.PersistentLogEntry)
			if aEntry.GetServerTimestamp().Equal(bEntry.GetServerTimestamp().Time) {
				if a.LogDescriptorID.String() == b.LogDescriptorID.String() {
					return aEntry.GetSeqNo() < bEntry.GetSeqNo()
				}
				return a.LogDescriptorID.String() < b.LogDescriptorID.String()
			}
			return aEntry.GetServerTimestamp().Before(bEntry.GetServerTimestamp().Time)
		})

		// We now have the ordered set of entries we expect the server should reproduce when
		// we ask for expanded logs of the parent log descriptor ID.
		var entries []*models.LogEntry
		for _, entry := range results {
			entries = append(entries, entry.LogEntry)
		}

		// Test reading merged data - read multiple times to ensure we're getting the same result each time e.g. no non-determinism in the line output order
		for i := 0; i < 10; i++ {
			expand := true
			reader, err := client.OpenLogReadStream(ctx, logDescriptor1.ID, &documents.LogSearchRequest{LogSearch: &models.LogSearch{Expand: &expand}})
			require.Nil(t, err)
			readRaw, err := ioutil.ReadAll(reader)
			require.Nil(t, err)
			reader.Close()
			var read []*models.LogEntry
			err = json.Unmarshal(readRaw, &read)
			require.Nil(t, err)
			require.True(t, structuredLogsEqual(entries, read), "attempt %d", i)
		}

		// Read back all descriptors after sealing - we should now see the end entry
		for _, descriptor := range descriptors {
			err = app.LogService.Seal(ctx, nil, descriptor.ID)
			require.Nil(t, err)
		}
		expand := true
		reader, err := client.OpenLogReadStream(ctx, logDescriptor1.ID, &documents.LogSearchRequest{LogSearch: &models.LogSearch{Expand: &expand}})
		require.Nil(t, err)
		readRaw, err := ioutil.ReadAll(reader)
		require.Nil(t, err)
		reader.Close()
		var read []*models.LogEntry
		err = json.Unmarshal(readRaw, &read)
		require.Nil(t, err)
		require.Equal(t, read[len(read)-1].Kind, models.LogEntryKindEnd)
		require.True(t, structuredLogsEqual(entries, read[:len(read)-1]))
	}
}

type entryWithDescriptorID struct {
	*models.LogEntry
	LogDescriptorID models.LogDescriptorID
}

func fillLogDescriptor(ctx context.Context, client *client.APIClient, descriptor *models.LogDescriptor) ([]*entryWithDescriptorID, error) {
	// Log entries have a server timestamp appended to them when they are received, which we can't know
	// ahead of time as a client. Here we write our entries, read them back, assert we get a 1:1 match
	// with what we wrote, and then use the read back entries further down in the test, enabling us to
	// sort using the server timestamp.
	var entries []*models.LogEntry
	for i := 0; i < 1000; i++ {
		entry := models.NewLogEntryLine(i+1, models.NewTime(time.Now()), fmt.Sprintf("%s: this is line %d", descriptor.ID, i+1), i+1, nil)
		entries = append(entries, entry)
	}
	writeData, err := json.Marshal(entries)
	if err != nil {
		return nil, fmt.Errorf("error marshaling JSON: %w", err)
	}
	writer, err := client.OpenLogWriteStream(ctx, descriptor.ID)
	if err != nil {
		return nil, fmt.Errorf("error opening write stream: %w", err)
	}
	_, err = writer.Write(writeData)
	if err != nil {
		return nil, fmt.Errorf("error writing to stream: %w", err)
	}
	writer.Close()
	reader, err := client.OpenLogReadStream(ctx, descriptor.ID, &documents.LogSearchRequest{LogSearch: &models.LogSearch{}})
	if err != nil {
		return nil, fmt.Errorf("error opening read stream: %w", err)
	}
	readData, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("error reading from stream: %w", err)
	}
	reader.Close()
	var read []*models.LogEntry
	err = json.Unmarshal(readData, &read)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}
	if !structuredLogsEqual(entries, read) {
		return nil, fmt.Errorf("error mismatch between data written and data read - see logs")
	}
	var withDescriptorID []*entryWithDescriptorID
	for _, entry := range read {
		withDescriptorID = append(withDescriptorID, &entryWithDescriptorID{LogDescriptorID: descriptor.ID, LogEntry: entry})
	}
	return withDescriptorID, nil
}

func rawLogsEqual(written []*models.LogEntry, read []byte) bool {
	var writtenBuf []byte
	for _, entry := range written {
		plaintext, ok := entry.Derived().(models.PlainTextLogEntry)
		if ok {
			writtenBuf = append(writtenBuf, plaintext.GetText()...)
			writtenBuf = append(writtenBuf, '\n')
		}
	}
	return bytes.Equal(writtenBuf, read)
}

func structuredLogsEqual(written []*models.LogEntry, read []*models.LogEntry) bool {
	if len(written) != len(read) {
		log.Printf("Len mismatch: %d vs %d", len(written), len(read))
		return false
	}
	for i := 0; i < len(written); i++ {
		w := written[i]
		r := read[i]
		if w.Kind != r.Kind {
			log.Printf("Type mismatch")
			return false
		}
		// NOTE: Intentionally not checking server timestamp as that is set by the server
		// and won't be present in the client-submitted log data.
		if reflect.TypeOf(w.Derived()) != reflect.TypeOf(r.Derived()) {
			log.Printf("Derived mismatch")
			return false
		}
		wd := w.Derived()
		rd := r.Derived()

		if wp, ok := wd.(models.PersistentLogEntry); ok {
			rp := rd.(models.PersistentLogEntry)
			if wp.GetSeqNo() != rp.GetSeqNo() {
				log.Printf("SeqNo mismatch idx %d: %d vs %d", i, wp.GetSeqNo(), rp.GetSeqNo())
				return false
			}
			if wp.GetClientTimestamp() != rp.GetClientTimestamp() {
				log.Printf("ClientTimestamp mismatch")
				return false
			}
		}

		if wt, ok := wd.(models.PlainTextLogEntry); ok {
			rt := rd.(models.PlainTextLogEntry)
			if wt.GetText() != rt.GetText() {
				log.Printf("Text mismatch")
				return false
			}
			if !((wt.GetParentBlockName() == nil) == (rt.GetParentBlockName() == nil)) {
				log.Printf("ParentBlockName mismatch: %s vs %s", wt.GetParentBlockName(), rt.GetParentBlockName())
				return false
			}
			if wt.GetParentBlockName() != nil && *wt.GetParentBlockName() != *rt.GetParentBlockName() {
				log.Printf("ParentBlockName value mismatch: %s vs %s", wt.GetParentBlockName(), rt.GetParentBlockName())
				return false
			}
		}

		switch wd.(type) {
		case *models.LogEntryBlock:
			wb := wd.(*models.LogEntryBlock)
			rb := rd.(*models.LogEntryBlock)
			if wb.Name != rb.Name {
				log.Printf("Name mismatch")
				return false
			}
		case *models.LogEntryLine:
			wl := wd.(*models.LogEntryLine)
			rl := rd.(*models.LogEntryLine)
			if wl.LineNo != rl.LineNo {
				log.Printf("LineNo mismatch")
				return false
			}
		default:
			log.Panicf("Unsupported log entry type: %T", w)
		}
	}
	return true
}

type basicReader struct {
	r io.Reader
}

func (r *basicReader) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}
