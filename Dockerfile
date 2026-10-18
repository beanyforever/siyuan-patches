# SiYuan's official image with its kernel rebuilt from the same release plus patches/.
# The kernel build matches upstream's Dockerfile; the front end is reused unchanged.
ARG SIYUAN_VERSION=v3.8.7

FROM golang:1.26-alpine AS kernel
ARG SIYUAN_VERSION
RUN apk add --no-cache gcc musl-dev git
WORKDIR /src
RUN git clone --quiet --depth 1 --branch "$SIYUAN_VERSION" --filter=blob:none --sparse \
        https://github.com/siyuan-note/siyuan.git . \
    && git sparse-checkout set kernel
COPY patches/ /patches/
RUN git apply /patches/*.patch
WORKDIR /src/kernel
ENV CGO_ENABLED=1
RUN --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg \
    go test -tags "fts5 sqlcipher" ./mcp/ \
    && go test -tags "fts5 sqlcipher" -run '^(TestFileToolWritesServeWhatTheToolWrote|TestFileToolCopyThroughALinkServesTheCopy|TestFileToolCopyIntoALinkServesTheCopy)$' ./mcp/tools/ \
    && go test -tags "fts5 sqlcipher" -run '^TestAttributeViewPropertyGuard|^TestSystemPageGuard|^TestSearchLeavesBlocksWhole|^(TestMoveToAnotherNotebookAndBackKeepsEdits|TestMoveAfterEditReadsNewType|TestRollbackDocHistoryRestoresAttributes|TestRollbackRepoSnapshotFileRestoresAttributes|TestRestoreRemovedBlockReadsNoAttributes|TestOlderPageAttributesStoredAfterASaveAreNotServed|TestJobAttributesAreNotServedBeforeItsPageIsWritten|TestFailedCommitCachesNothing|TestNewRowPageReadsWithItsDatabase|TestGroupedTableCellEditKeepsStaticText|TestDocToHeadingServesTheHeadingsAttributes|TestDocToHeadingLeavesNoCopyOfTheSourcePage|TestDataImportServesTheImportedCopies|TestSyncDownloadServesTheDownloadedPage|TestCheckoutServesTheSnapshotRightAfterItsWrite|TestDoc2HeadingLeavesNoCurrentCopyOfItsSourcePage|TestRollingAnUnchangedPageBackToASnapshotKeepsItsFile|TestSyncServesTheMergedPageWhileItUploads|TestSyncServesNoRemovedPageWhileItUploads|TestTripwireLogsAPageSaveOutsideATurnOncePerSite|TestTripwireLogsADatabaseSaveOutsideATurnOncePerSite|TestTripwireLogsSetBlockAttrsOnce|TestTripwireLeavesSavesInATurnUnlogged|TestTripwireLeavesTheFirstSaveOfANewFileUnlogged|TestTripwireLeavesRepairsWhileLoadingUnlogged|TestTripwireTheFlushersEmptyTickIsNoTurn|TestTripwireLogsOnlyTheSaveMadeOutsideATurn|TestTripwireChallengeNamesEachWriterBehindAvSaveView|TestTripwireChallengeLogsASaveToAPathItsPageHasLeft|TestTripwireChallengeNamesAGoStatementThatCallsASaveFunction|TestTripwireProbeNamesTheWriterThroughEveryWrapper|TestTripwireProbeAPanickingJobLeavesNoTurn)$' ./model/ \
    && go test -tags "fts5 sqlcipher" -run '^(TestReadCheckRefusesCopiesOlderThanTheirPage|TestBlockAttributesWithoutTheirPageAreNotServed|TestDroppingBlockAttributesRaisesTheirPage|TestClearingACacheMakesOnlyItsEntriesMiss|TestForgettingVersionsKeepsOtherEntries|TestFillVersionsForgetPastTheirLimit|TestConcurrentRaisesNeverLowerTheFloor|TestForgettingVersionsStallsReadsBriefly|TestOverlappingClearsKeepTheLaterFloor|TestWriteEndsCacheOnlyWithoutAnotherRaise|TestRaisingAPageMakesOnlyItsCopiesMiss|TestRaisingADatabaseMakesOnlyItsCopiesMiss|TestRaisingADatabaseNeverReadAddsNoVersion|TestRaisingEverythingRefusesEveryOlderCopy|TestWritesOverlappingARaiseCacheNothing|TestRaisingWhatChangedSinceATokenLeavesTheRest|TestRaisingWhatChangedSinceCoversARaiseOfEverything)$' ./cache/ \
    && go test -tags "fts5 sqlcipher" -run '^(TestTreeFillRacesWrite|TestTreeFillsRacingASaveServeTheSave|TestOverlappingWritesOfAPageCacheWhatTheFileHolds|TestPanickingPageWriteRaisesItsPageAndReleasesItsLocks|TestDataPathsNameWhatTheyHold|TestRaiseForDataPathRaisesWhatAFolderHolds|TestBeginDataPathWriteRaisesWhatARemoveTookAway|TestDataCopiesNameWhatTheyWrote|TestRemovingAFolderRaisesAPageWrittenIntoItDuringTheRemove|TestRemovingAFolderRaisesAPageABulkWriterPutInItDuringTheRemove|TestRemovingALargeFolderRaisesAPageWrittenIntoItDuringTheRemove|TestTripwireLogsAPageSaveOutsideATurnAtItsWriter|TestTripwirePassesOverRepairsMadeWhileLoading)$' ./filesys/ \
    && go test -tags "fts5 sqlcipher" -run '^(TestBatchGetBlockAttrsWitTreesCachesNothing|TestBlockAttrsReadsFillWhatTheyLoad|TestBlockAttrsFillThenWriteKeepsWrite|TestPageWritesMakeOlderBlockAttrsMiss|TestBlockAttrsReadBeforeTheirPageChangedAreNotServed|TestKanbanRenderAfterAWriteServesNewAttributes)$' ./sql/ \
    && go test -tags "fts5 sqlcipher" -run '^(TestAVReadAfterSaveGivesTheSave|TestAVSaveOfUnchangedCopyRacingASaveKeepsTheSave|TestAVHistoryReadLeavesTheLiveEntry|TestAVDropRacingAFillInANotebookLeavesNoOlderCopy|TestAVOverlappingSavesCacheWhatTheFileHolds|TestAVSaveWithoutOverlapIsServedFromTheCache|TestPanickingDatabaseSaveRaisesItsVersionAndReleasesItsLocks|TestTripwireLogsADatabaseSaveOutsideATurnAtItsWriter)$' ./av/ \
    && go test -tags "fts5 sqlcipher" ./tripwire/ \
    && go test -tags "fts5 sqlcipher" -run '^(TestPutFileServesTheFileItWrote|TestCopiedFilesServeTheCopy|TestRenamedFilesServeTheFileRenamedIn|TestRemovedPageServesNoAttributes|TestUnzippedFilesServeTheArchivesCopy)$' ./api/ \
    && go build -tags "fts5 sqlcipher" -ldflags "-s -w"

FROM b3log/siyuan:${SIYUAN_VERSION}
COPY --from=kernel --chmod=755 /src/kernel/kernel /opt/siyuan/kernel
