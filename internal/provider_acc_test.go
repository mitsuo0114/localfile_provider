package internal

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"local-file": providerserver.NewProtocol6WithError(NewProvider("test")),
}

func testAccTxtResourceConfig(baseDir, data string) string {
	return fmt.Sprintf(`
provider "local-file" {
  base_dir = "%s"
}

resource "local-file_txt" "test" {
  name = "acc.txt"
  data = "%s"
}
`, baseDir, data)
}

func TestAccTxtResource_basic(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTxtResourceConfig(tempDir, "hello"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("local-file_txt.test", "data", "hello"),
				),
			},
			{
				Config: testAccTxtResourceConfig(tempDir, "updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("local-file_txt.test", "data", "updated"),
				),
			},
		},
	})
}

func testAccTxtDataSourceConfig(baseDir string) string {
	return fmt.Sprintf(`
provider "local-file" {
  base_dir = "%s"
}

resource "local-file_txt" "write" {
  name = "data.txt"
  data = "from resource"
}

data "local-file_txt" "read" {
  name = local-file_txt.write.name
}
`, baseDir)
}

func TestAccTxtDataSource_basic(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTxtDataSourceConfig(tempDir),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.local-file_txt.read", "data", "from resource"),
					resource.TestCheckResourceAttr("local-file_txt.write", "data", "from resource"),
				),
			},
		},
	})
}
