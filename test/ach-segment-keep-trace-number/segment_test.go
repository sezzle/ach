// Licensed to The Moov Authors under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. The Moov Authors licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package main

import (
	"testing"
	"time"

	"github.com/moov-io/ach"
)

func TestSegmentKeepTraceNumber(t *testing.T) {
	// To create a file
	fh := ach.NewFileHeader()
	fh.ImmediateDestination = "231380104"
	fh.ImmediateOrigin = "121042882"
	fh.FileCreationDate = time.Now().Format("060102")
	fh.ImmediateDestinationName = "Federal Reserve Bank"
	fh.ImmediateOriginName = "My Bank Name"
	file := ach.NewFile()
	file.SetHeader(fh)

	// To create a batch.
	// Errors only if payment type is not supported.
	bh := ach.NewBatchHeader()
	bh.ServiceClassCode = ach.MixedDebitsAndCredits
	bh.CompanyName = "Your Company"
	bh.CompanyIdentification = file.Header.ImmediateOrigin
	bh.StandardEntryClassCode = ach.PPD
	bh.CompanyEntryDescription = "Trans. Description"
	bh.EffectiveEntryDate = time.Now().AddDate(0, 0, 1).Format("060102") // YYMMDD
	bh.ODFIIdentification = "121042882"

	batch, err := ach.NewBatch(bh)
	if err != nil {
		t.Fatalf("%T: %v", err, err)
	}
	batch.SetValidation(&ach.ValidateOpts{
		BypassOriginValidation: true,
	})

	// To create an entry
	entry := ach.NewEntryDetail()
	entry.TransactionCode = ach.CheckingCredit
	entry.SetRDFI("231380104")
	entry.DFIAccountNumber = "81967038518"
	entry.Amount = 1000000
	entry.IndividualName = "Wade Arnold"
	entry.SetTraceNumber("12345678", 1)
	entry.IdentificationNumber = "ABC##jvkdjfuiwn"
	entry.Category = ach.CategoryForward
	entry.AddendaRecordIndicator = 1

	// Add one or more optional addenda records for an entry
	addenda := ach.NewAddenda05()
	addenda.PaymentRelatedInformation = "bonus pay for amazing work on #OSS"
	entry.AddAddenda05(addenda)

	// Entries are added to batches like so:
	batch.AddEntry(entry)

	// When all of the Entries are added to the batch we must create it.
	if err := batch.Create(); err != nil {
		t.Fatalf("%T: %v", err, err)
	}

	// And batches are added to files much the same way:
	file.AddBatch(batch)

	// Now add a new batch for accepting payments on the web
	bh2 := ach.NewBatchHeader()
	bh2.ServiceClassCode = ach.DebitsOnly
	bh2.CompanyName = "Your Company"
	bh2.CompanyIdentification = file.Header.ImmediateOrigin
	bh2.StandardEntryClassCode = ach.WEB
	bh2.CompanyEntryDescription = "Subscribe"
	bh2.EffectiveEntryDate = time.Now().AddDate(0, 0, 1).Format("060102") // YYMMDD
	bh2.ODFIIdentification = "121042882"

	batch2, err := ach.NewBatch(bh2)
	if err != nil {
		t.Fatalf("%T: %v", err, err)
	}
	batch2.SetValidation(&ach.ValidateOpts{
		BypassOriginValidation: true,
	})

	// Add an entry and define if it is a single or recurring payment
	// The following is a recurring payment for $7.99
	entry2 := ach.NewEntryDetail()
	entry2.TransactionCode = ach.CheckingDebit
	entry2.SetRDFI("231380104")
	entry2.DFIAccountNumber = "81967038518"
	entry2.Amount = 799
	entry2.IndividualName = "Wade Arnold"
	entry2.SetTraceNumber("87654321", 2)
	entry2.IdentificationNumber = "#123456"
	entry2.DiscretionaryData = "R"
	entry2.Category = ach.CategoryForward
	entry2.AddendaRecordIndicator = 1

	// To add one or more optional addenda records for an entry
	addenda2 := ach.NewAddenda05()
	addenda2.PaymentRelatedInformation = "Monthly Membership Subscription"
	entry2.AddAddenda05(addenda2)

	// Add the entry to the batch
	batch2.AddEntry(entry2)

	// Create and add the second batch
	if err := batch2.Create(); err != nil {
		t.Fatalf("%T: %v", err, err)
	}
	file.AddBatch(batch2)

	// Once we've added all our batches we must create the file
	if err := file.Create(); err != nil {
		t.Fatalf("%T: %v", err, err)
	}

	creditFile, debitFile, err := file.SegmentFile(nil)
	if err != nil {
		t.Fatal(err)
	}

	if entries := creditFile.Batches[0].GetEntries(); entries[0].TraceNumber != entry.TraceNumber {
		t.Fatalf("TraceNumber before=%v after=%v", entry.TraceNumber, entries[0].TraceNumber)
	}
	if entries := debitFile.Batches[0].GetEntries(); entries[0].TraceNumber != entry2.TraceNumber {
		t.Fatalf("TraceNumber before=%v after=%v", entry.TraceNumber, entries[0].TraceNumber)
	}

	// Blank out TraceNumbers and let sequential ones be created
	batch.SetValidation(nil)
	entry.TraceNumber = ""
	if err := batch.Create(); err != nil {
		t.Fatal(err)
	}

	entry2.TraceNumber = ""
	batch2.SetValidation(nil)
	if err := batch2.Create(); err != nil {
		t.Fatal(err)
	}

	if err := file.Create(); err != nil {
		t.Fatalf("%T: %v", err, err)
	}

	creditFile, debitFile, err = file.SegmentFile(nil)
	if err != nil {
		t.Fatalf("%T: %v", err, err)
	}

	if entries := creditFile.Batches[0].GetEntries(); entries[0].TraceNumber == "" || entries[0].TraceNumber != "121042880000001" {
		t.Fatalf("TraceNumber before=%v after=%v", entry.TraceNumber, entries[0].TraceNumber)
	}
	if entries := debitFile.Batches[0].GetEntries(); entries[0].TraceNumber == "" || entries[0].TraceNumber != "121042880000001" {
		t.Fatalf("TraceNumber before=%v after=%v", entry.TraceNumber, entries[0].TraceNumber)
	}
}
