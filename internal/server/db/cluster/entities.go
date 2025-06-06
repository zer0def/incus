package cluster

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/lxc/incus/v6/internal/version"
)

// Numeric type codes identifying different kind of entities.
const (
	TypeContainer             = 0
	TypeImage                 = 1
	TypeProfile               = 2
	TypeProject               = 3
	TypeCertificate           = 4
	TypeInstance              = 5
	TypeInstanceBackup        = 6
	TypeInstanceSnapshot      = 7
	TypeNetwork               = 8
	TypeNetworkACL            = 9
	TypeNode                  = 10
	TypeOperation             = 11
	TypeStoragePool           = 12
	TypeStorageVolume         = 13
	TypeStorageVolumeBackup   = 14
	TypeStorageVolumeSnapshot = 15
	TypeWarning               = 16
	TypeClusterGroup          = 17
	TypeStorageBucket         = 18
)

// EntityNames associates an entity code to its name.
var EntityNames = map[int]string{
	TypeCertificate:           "certificate",
	TypeClusterGroup:          "cluster group",
	TypeContainer:             "container",
	TypeImage:                 "image",
	TypeInstanceBackup:        "instance backup",
	TypeInstance:              "instance",
	TypeInstanceSnapshot:      "instance snapshot",
	TypeNetworkACL:            "network acl",
	TypeNetwork:               "network",
	TypeNode:                  "node",
	TypeOperation:             "operation",
	TypeProfile:               "profile",
	TypeProject:               "project",
	TypeStorageBucket:         "storage bucket",
	TypeStoragePool:           "storage pool",
	TypeStorageVolumeBackup:   "storage volume backup",
	TypeStorageVolumeSnapshot: "storage volume snapshot",
	TypeStorageVolume:         "storage volume",
	TypeWarning:               "warning",
}

// EntityTypes associates an entity name to its type code.
var EntityTypes = map[string]int{}

// EntityURIs associates an entity code to its URI pattern.
var EntityURIs = map[int]string{
	TypeCertificate:           "/" + version.APIVersion + "/certificates/%s",
	TypeClusterGroup:          "/" + version.APIVersion + "/cluster/groups/%s",
	TypeContainer:             "/" + version.APIVersion + "/containers/%s?project=%s",
	TypeImage:                 "/" + version.APIVersion + "/images/%s?project=%s",
	TypeInstanceBackup:        "/" + version.APIVersion + "/instances/%s/backups/%s?project=%s",
	TypeInstanceSnapshot:      "/" + version.APIVersion + "/instances/%s/snapshots/%s?project=%s",
	TypeInstance:              "/" + version.APIVersion + "/instances/%s?project=%s",
	TypeNetworkACL:            "/" + version.APIVersion + "/network-acls/%s?project=%s",
	TypeNetwork:               "/" + version.APIVersion + "/networks/%s?project=%s",
	TypeNode:                  "/" + version.APIVersion + "/cluster/members/%s",
	TypeOperation:             "/" + version.APIVersion + "/operations/%s",
	TypeProfile:               "/" + version.APIVersion + "/profiles/%s?project=%s",
	TypeProject:               "/" + version.APIVersion + "/projects/%s",
	TypeStorageBucket:         "/" + version.APIVersion + "/storage-pools/%s/buckets/%s?project=%s",
	TypeStoragePool:           "/" + version.APIVersion + "/storage-pools/%s",
	TypeStorageVolumeBackup:   "/" + version.APIVersion + "/storage-pools/%s/volumes/%s/%s/backups/%s?project=%s",
	TypeStorageVolumeSnapshot: "/" + version.APIVersion + "/storage-pools/%s/volumes/%s/%s/snapshots/%s?project=%s",
	TypeStorageVolume:         "/" + version.APIVersion + "/storage-pools/%s/volumes/%s/%s?project=%s",
	TypeWarning:               "/" + version.APIVersion + "/warnings/%s",
}

func init() {
	for code, name := range EntityNames {
		EntityTypes[name] = code
	}
}

// URLToEntityType parses a raw URL string and returns the entity type, the project, the location and the path arguments. The
// returned project is set to "default" if it is not present (unless the entity type is TypeProject, in which case it is
// set to the value of the path parameter). An error is returned if the URL is not recognised.
func URLToEntityType(rawURL string) (int, string, string, []string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return -1, "", "", nil, fmt.Errorf("Failed to parse url %q into an entity type: %w", rawURL, err)
	}

	// We need to space separate the path because fmt.Sscanf uses this as a delimiter.
	spaceSeparatedURLPath := strings.ReplaceAll(u.Path, "/", " / ")
	for entityType, entityURI := range EntityURIs {
		entityPath, _, _ := strings.Cut(entityURI, "?")

		// Skip if we don't have the same number of slashes.
		if strings.Count(entityPath, "/") != strings.Count(u.Path, "/") {
			continue
		}

		spaceSeparatedEntityPath := strings.ReplaceAll(entityPath, "/", " / ")

		// Make an []any for the number of expected path arguments and set each value in the slice to a *string.
		nPathArgs := strings.Count(spaceSeparatedEntityPath, "%s")
		pathArgsAny := make([]any, 0, nPathArgs)
		for range nPathArgs {
			var pathComponentStr string
			pathArgsAny = append(pathArgsAny, &pathComponentStr)
		}

		// Scan the given URL into the entity URL. If we found all the expected path arguments and there
		// are no errors we have a match.
		nFound, err := fmt.Sscanf(spaceSeparatedURLPath, spaceSeparatedEntityPath, pathArgsAny...)
		if nFound == nPathArgs && err == nil {
			pathArgs := make([]string, 0, nPathArgs)
			for _, pathArgAny := range pathArgsAny {
				pathArgPtr := pathArgAny.(*string)
				pathArgs = append(pathArgs, *pathArgPtr)
			}

			projectName := u.Query().Get("project")
			if projectName == "" {
				projectName = "default"
			}

			location := u.Query().Get("target")

			if entityType == TypeProject {
				return TypeProject, pathArgs[0], location, pathArgs, nil
			}

			return entityType, projectName, location, pathArgs, nil
		}
	}

	return -1, "", "", nil, fmt.Errorf("Unknown entity URL %q", u.String())
}
