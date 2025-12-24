package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

type SaveBlueprintFixture struct {
	InstanceState *state.InstanceState
	Update        bool
}

func SetupSaveBlueprintFixtures(dirPath string, updates []int) (map[int]SaveBlueprintFixture, error) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	saveBlueprintFixtures := make(map[int]SaveBlueprintFixture)
	for i := 1; i <= len(dirEntries); i++ {
		isUpdate := slices.Contains(updates, i)
		fixture, err := loadSaveBlueprintFixture(i, dirPath, isUpdate)
		if err != nil {
			return nil, err
		}
		saveBlueprintFixtures[i] = fixture
	}

	return saveBlueprintFixtures, nil
}

func loadSaveBlueprintFixture(fixtureNumber int, dirPath string, isUpdate bool) (SaveBlueprintFixture, error) {
	fileName := fixtureFileName(fixtureNumber)
	filePath := path.Join(dirPath, fileName)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return SaveBlueprintFixture{}, err
	}

	instanceState := &state.InstanceState{}
	err = json.Unmarshal(data, instanceState)
	if err != nil {
		return SaveBlueprintFixture{}, err
	}

	return SaveBlueprintFixture{
		InstanceState: instanceState,
		Update:        isUpdate,
	}, nil
}

type SaveResourceFixture struct {
	ResourceState *state.ResourceState
	Update        bool
}

func SetupSaveResourceFixtures(dirPath string, updates []int) (map[int]SaveResourceFixture, error) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	saveResourceFixtures := make(map[int]SaveResourceFixture)
	for i := 1; i <= len(dirEntries); i++ {
		isUpdate := slices.Contains(updates, i)
		fixture, err := loadSaveResourceFixture(i, isUpdate)
		if err != nil {
			return nil, err
		}
		saveResourceFixtures[i] = fixture
	}

	return saveResourceFixtures, nil
}

func loadSaveResourceFixture(fixtureNumber int, isUpdate bool) (SaveResourceFixture, error) {
	fileName := fixtureFileName(fixtureNumber)
	filePath := path.Join(saveInputDir(), "resources", fileName)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return SaveResourceFixture{}, err
	}

	resourceState := &state.ResourceState{}
	err = json.Unmarshal(data, resourceState)
	if err != nil {
		return SaveResourceFixture{}, err
	}

	return SaveResourceFixture{
		ResourceState: resourceState,
		Update:        isUpdate,
	}, nil
}

type SaveResourceDriftFixture struct {
	DriftState *state.ResourceDriftState
	Update     bool
}

func SetupSaveResourceDriftFixtures(dirPath string, updates []int) (map[int]SaveResourceDriftFixture, error) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	saveResourceDriftFixtures := make(map[int]SaveResourceDriftFixture)
	for i := 1; i <= len(dirEntries); i++ {
		isUpdate := slices.Contains(updates, i)
		fixture, err := loadSaveResourceDriftFixture(i, isUpdate)
		if err != nil {
			return nil, err
		}
		saveResourceDriftFixtures[i] = fixture
	}

	return saveResourceDriftFixtures, nil
}

func loadSaveResourceDriftFixture(fixtureNumber int, isUpdate bool) (SaveResourceDriftFixture, error) {
	fileName := fixtureFileName(fixtureNumber)
	filePath := path.Join(saveInputDir(), "resource-drift", fileName)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return SaveResourceDriftFixture{}, err
	}

	driftState := &state.ResourceDriftState{}
	err = json.Unmarshal(data, driftState)
	if err != nil {
		return SaveResourceDriftFixture{}, err
	}

	return SaveResourceDriftFixture{
		DriftState: driftState,
		Update:     isUpdate,
	}, nil
}

type SaveLinkFixture struct {
	LinkState *state.LinkState
	Update    bool
}

type SaveLinkDriftFixture struct {
	DriftState *state.LinkDriftState
	Update     bool
}

func SetupSaveLinkDriftFixtures(dirPath string, updates []int) (map[int]SaveLinkDriftFixture, error) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	saveLinkDriftFixtures := make(map[int]SaveLinkDriftFixture)
	for i := 1; i <= len(dirEntries); i++ {
		isUpdate := slices.Contains(updates, i)
		fixture, err := loadSaveLinkDriftFixture(i, dirPath, isUpdate)
		if err != nil {
			return nil, err
		}
		saveLinkDriftFixtures[i] = fixture
	}

	return saveLinkDriftFixtures, nil
}

func loadSaveLinkDriftFixture(fixtureNumber int, dirPath string, isUpdate bool) (SaveLinkDriftFixture, error) {
	fileName := fixtureFileName(fixtureNumber)
	filePath := path.Join(dirPath, fileName)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return SaveLinkDriftFixture{}, err
	}

	driftState := &state.LinkDriftState{}
	err = json.Unmarshal(data, driftState)
	if err != nil {
		return SaveLinkDriftFixture{}, err
	}

	return SaveLinkDriftFixture{
		DriftState: driftState,
		Update:     isUpdate,
	}, nil
}

func SetupSaveLinkFixtures(dirPath string, updates []int) (map[int]SaveLinkFixture, error) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	saveLinkFixtures := make(map[int]SaveLinkFixture)
	for i := 1; i <= len(dirEntries); i++ {
		isUpdate := slices.Contains(updates, i)
		fixture, err := loadSaveLinkFixture(i, isUpdate)
		if err != nil {
			return nil, err
		}
		saveLinkFixtures[i] = fixture
	}

	return saveLinkFixtures, nil
}

func loadSaveLinkFixture(fixtureNumber int, isUpdate bool) (SaveLinkFixture, error) {
	fileName := fmt.Sprintf("%d.json", fixtureNumber)
	filePath := path.Join(saveInputDir(), "links", fileName)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return SaveLinkFixture{}, err
	}

	linkState := &state.LinkState{}
	err = json.Unmarshal(data, linkState)
	if err != nil {
		return SaveLinkFixture{}, err
	}

	return SaveLinkFixture{
		LinkState: linkState,
		Update:    isUpdate,
	}, nil
}

type SaveEventFixture struct {
	Event *manage.Event
}

func SetupSaveEventFixtures(dirPath string) (map[int]SaveEventFixture, error) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	saveEventFixtures := make(map[int]SaveEventFixture)
	for i := 1; i <= len(dirEntries); i++ {
		fixture, err := loadSaveEventFixture(i)
		if err != nil {
			return nil, err
		}
		saveEventFixtures[i] = fixture
	}

	return saveEventFixtures, nil
}

func loadSaveEventFixture(fixtureNumber int) (SaveEventFixture, error) {
	fileName := fmt.Sprintf("%d.json", fixtureNumber)
	filePath := path.Join(saveInputDir(), "events", fileName)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return SaveEventFixture{}, err
	}

	event := &manage.Event{}
	err = json.Unmarshal(data, event)
	if err != nil {
		return SaveEventFixture{}, err
	}

	return SaveEventFixture{
		Event: event,
	}, nil
}

func CreateEventStreamSaveFixtures(
	channelType string,
	channelID string,
	streamFixtureEventIDs []uuid.UUID,
) ([]SaveEventFixture, error) {
	// Sleep between preparing each fixture to ensure the UUIDs contain different
	// timestamps to millisecond precision to assert that the events are
	// streamed in the correct order.
	fixtures := make([]SaveEventFixture, len(streamFixtureEventIDs))
	for i := 0; i < len(streamFixtureEventIDs); i += 1 {
		id := streamFixtureEventIDs[i]

		fixtures[i] = SaveEventFixture{
			Event: &manage.Event{
				ID:          id.String(),
				Type:        "resource",
				ChannelType: channelType,
				ChannelID:   channelID,
				Data:        fmt.Sprintf("{\"value\":\"%d\"}", i),
				Timestamp:   time.Now().Unix(),
			},
		}
		time.Sleep(5 * time.Millisecond)
	}

	return fixtures, nil
}

// UUIDv7 values for event IDs in timestamp order.
// These should be used in conjunction with the stream for the channel:
// "changesets_db58eda8-36c6-4180-a9cb-557f3392361c".
var StreamFixtureEventIDs1 = []uuid.UUID{
	uuid.MustParse("01966574-33ba-73c4-a5c0-a0b55249d39a"),
	uuid.MustParse("01966574-69ef-7b02-81cd-7fdbbbead77d"),
	uuid.MustParse("01966574-a5fe-7b22-9229-12a19afc8c32"),
	uuid.MustParse("01966575-47f8-7770-8a3c-56ea2e2b8dee"),
	uuid.MustParse("01966575-7ce6-7923-be83-011cebc8c8d3"),
	uuid.MustParse("01966575-a91e-7829-9f74-5069446071bf"),
	uuid.MustParse("01966576-0654-7f14-be3b-6af31cd6a1f5"),
	uuid.MustParse("01966576-368a-7a53-9f4e-38f9a5ef8ece"),
	uuid.MustParse("01966576-78b4-7711-9d4a-929e8dc29eb6"),
	uuid.MustParse("01966576-a6b5-7e37-8bf2-60e0eb10602e"),
	uuid.MustParse("01966576-e3e3-717d-bc25-324c29056a2f"),
	uuid.MustParse("01966577-3210-7562-8f1e-5a85200907b8"),
	uuid.MustParse("01966577-65f9-7cbb-ae74-93c7766c7d80"),
	uuid.MustParse("01966577-bff2-7829-b14b-7041be6c56b5"),
	uuid.MustParse("01966577-f76b-73b0-ae60-64d241ce4e8a"),
	uuid.MustParse("01966578-4544-729f-b968-b5893ea9fbdc"),
	uuid.MustParse("01966578-7675-7004-820f-d85b3e7616a7"),
	uuid.MustParse("01966578-acbf-735c-a318-6393dc267599"),
	uuid.MustParse("01966578-fe83-7cbd-8790-1a93dbf62e18"),
	uuid.MustParse("01966579-28d4-7c43-b4b2-f29238540587"),
}

// UUIDv7 values for event IDs in timestamp order.
// These should be used in conjunction with the stream for the channel:
// "changesets_eabba2f8-5c74-4c51-a068-b340f718314a".
var StreamFixtureEventIDs2 = []uuid.UUID{
	uuid.MustParse("0124e053-3580-7000-af52-4ac381dd6b44"),
	uuid.MustParse("0124e053-3582-7000-8206-d47b8e228491"),
	uuid.MustParse("0124e053-3584-7000-890a-36c56957891e"),
	uuid.MustParse("0124e053-3586-7000-9655-2786c4fe6265"),
	uuid.MustParse("0124e053-3588-7000-879b-ecdfab5040fb"),
	uuid.MustParse("0124e053-358a-7000-a35b-4e89ad94ea3b"),
	uuid.MustParse("0124e053-358c-7000-811b-f6ea3ae88425"),
	uuid.MustParse("0124e053-358e-7000-b5f0-4312eb6182f0"),
	uuid.MustParse("0124e053-3590-7000-9909-509308ef9a23"),
	uuid.MustParse("0124e053-3592-7000-846d-4aef540b6464"),
	uuid.MustParse("0124e053-3594-7000-8ed0-be0e4f5da934"),
	uuid.MustParse("0124e053-3596-7000-9336-8c37401e324c"),
	uuid.MustParse("0124e053-3598-7000-afa5-d0eecccba403"),
	uuid.MustParse("0124e053-359a-7000-a136-99afc1b30c9c"),
	uuid.MustParse("0124e053-359c-7000-9e42-48c82b6fe8eb"),
	uuid.MustParse("0124e053-359e-7000-8915-e35a14b642ba"),
	uuid.MustParse("0124e053-35a0-7000-90d6-e95020933403"),
	uuid.MustParse("0124e053-35a2-7000-8497-c67c6f11d926"),
	uuid.MustParse("0124e053-35a4-7000-8d69-12e2ac28ffb0"),
	uuid.MustParse("0124e053-35a6-7000-b03c-bb1fa7943039"),
}

type SaveChangesetFixture struct {
	Changeset *manage.Changeset
}

func SetupSaveChangesetFixtures(dirPath string) (map[int]SaveChangesetFixture, error) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	saveChangesetFixtures := make(map[int]SaveChangesetFixture)
	for i := 1; i <= len(dirEntries); i++ {
		fixture, err := loadSaveChangesetFixture(i)
		if err != nil {
			return nil, err
		}
		saveChangesetFixtures[i] = fixture
	}

	return saveChangesetFixtures, nil
}

func loadSaveChangesetFixture(fixtureNumber int) (SaveChangesetFixture, error) {
	fileName := fmt.Sprintf("%d.json", fixtureNumber)
	filePath := path.Join(saveInputDir(), "changesets", fileName)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return SaveChangesetFixture{}, err
	}

	changeset := &manage.Changeset{}
	err = json.Unmarshal(data, changeset)
	if err != nil {
		return SaveChangesetFixture{}, err
	}

	return SaveChangesetFixture{
		Changeset: changeset,
	}, nil
}

type SaveBlueprintValidationFixture struct {
	Validation *manage.BlueprintValidation
}

func SetupSaveBlueprintValidationFixtures(
	dirPath string,
) (map[int]SaveBlueprintValidationFixture, error) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	saveBlueprintValidationFixtures := make(map[int]SaveBlueprintValidationFixture)
	for i := 1; i <= len(dirEntries); i++ {
		fixture, err := loadSaveBlueprintValidationFixture(i)
		if err != nil {
			return nil, err
		}
		saveBlueprintValidationFixtures[i] = fixture
	}

	return saveBlueprintValidationFixtures, nil
}

func loadSaveBlueprintValidationFixture(fixtureNumber int) (SaveBlueprintValidationFixture, error) {
	fileName := fmt.Sprintf("%d.json", fixtureNumber)
	filePath := path.Join(saveInputDir(), "blueprint-validations", fileName)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return SaveBlueprintValidationFixture{}, err
	}

	validation := &manage.BlueprintValidation{}
	err = json.Unmarshal(data, validation)
	if err != nil {
		return SaveBlueprintValidationFixture{}, err
	}

	return SaveBlueprintValidationFixture{
		Validation: validation,
	}, nil
}

type SaveReconciliationResultFixture struct {
	ReconciliationResult *manage.ReconciliationResult
}

func SetupSaveReconciliationResultFixtures(dirPath string) (map[int]SaveReconciliationResultFixture, error) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	saveReconciliationResultFixtures := make(map[int]SaveReconciliationResultFixture)
	for i := 1; i <= len(dirEntries); i++ {
		fixture, err := loadSaveReconciliationResultFixture(i, dirPath)
		if err != nil {
			return nil, err
		}
		saveReconciliationResultFixtures[i] = fixture
	}

	return saveReconciliationResultFixtures, nil
}

func loadSaveReconciliationResultFixture(fixtureNumber int, dirPath string) (SaveReconciliationResultFixture, error) {
	fileName := fmt.Sprintf("%d.json", fixtureNumber)
	filePath := path.Join(dirPath, fileName)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return SaveReconciliationResultFixture{}, err
	}

	reconciliationResult := &manage.ReconciliationResult{}
	err = json.Unmarshal(data, reconciliationResult)
	if err != nil {
		return SaveReconciliationResultFixture{}, err
	}

	return SaveReconciliationResultFixture{
		ReconciliationResult: reconciliationResult,
	}, nil
}

func saveInputDir() string {
	return path.Join("__testdata", "save-input")
}

func fixtureFileName(fixtureNumber int) string {
	return fmt.Sprintf("%d.json", fixtureNumber)
}
