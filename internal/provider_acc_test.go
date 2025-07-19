package internal

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	ProviderTypeName: providerserver.NewProtocol6WithError(NewProvider("test")),
}

func testAccTxtResourceConfig(baseDir, data string) string {
	return fmt.Sprintf(`
provider "%s" {
  base_dir = "%s"
}

resource "%s_txt" "test" {
  name = "acc.txt"
  data = "%s"
}
`, ProviderTypeName, baseDir, ProviderTypeName, data)
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
					resource.TestCheckResourceAttr(fmt.Sprintf("%s_txt.test", ProviderTypeName), "data", "hello"),
				),
			},
			{
				Config: testAccTxtResourceConfig(tempDir, "updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("%s_txt.test", ProviderTypeName), "data", "updated"),
				),
			},
		},
	})
}

func testAccTxtDataSourceConfig(baseDir string) string {
	return fmt.Sprintf(`
provider "%s" {
  base_dir = "%s"
}

resource "%s_txt" "write" {
  name = "data.txt"
  data = "from resource"
}

data "%s_txt" "read" {
  name = %s_txt.write.name
}
`, ProviderTypeName, baseDir, ProviderTypeName, ProviderTypeName, ProviderTypeName)
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
					resource.TestCheckResourceAttr(fmt.Sprintf("data.%s_txt.read", ProviderTypeName), "data", "from resource"),
					resource.TestCheckResourceAttr(fmt.Sprintf("%s_txt.write", ProviderTypeName), "data", "from resource"),
				),
			},
		},
	})
}
