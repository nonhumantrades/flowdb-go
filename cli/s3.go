package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nonhumantrades/flowdb-go/types"
)

func (c *Cli) handleS3Help(_ *S3Help) {
	fmt.Println("S3 commands:")
	fmt.Println("  s3 list")
	fmt.Println("      List all configured S3 profiles (name, bucket, url, partial access key).")
	fmt.Println()
	fmt.Println("  s3 add [name=<profile-name>]")
	fmt.Println("      Add or update an S3 profile. If no name is given, you will be prompted.")
	fmt.Println("      You will be asked for bucket, endpoint URL, region, access key, secret key.")
	fmt.Println()
	fmt.Println("  s3 delete [name=<profile-name>]")
	fmt.Println("      Delete an S3 profile. If no name is given, you can select it interactively.")
	fmt.Println("      You will be asked to confirm before deletion.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  s3 list")
	fmt.Println("  s3 add name=prod-backups")
	fmt.Println("  s3 delete name=old-backups")
	fmt.Println("  s3 delete            # choose from list")
}

func (c *Cli) handleS3List(cmd *S3List) {
	if len(c.state.S3Profiles) == 0 {
		fmt.Println("no S3 profiles configured")
		return
	}

	fmt.Printf("%-3s %-16s %-20s %-32s %-20s\n", "#", "NAME", "BUCKET", "URL", "ACCESS_KEY")
	for i, p := range c.state.S3Profiles {
		fmt.Printf("%-3d %-16s %-20s %-32s %-20s\n",
			i+1,
			p.Name,
			p.Creds.Bucket,
			p.Creds.Url,
			maskKey(p.Creds.AccessKey),
		)
	}
}

func (c *Cli) handleS3Add(cmd *S3Add) {
	name := strings.TrimSpace(cmd.Name)
	var err error
	if name == "" {
		name, err = c.promptNonEmpty("Profile name")
		if err != nil {
			fmt.Printf("aborted: %v\n", err)
			return
		}
	}

	idx := c.findS3ProfileIndex(name)
	var existing *types.S3Credentials
	if idx >= 0 {
		existing = &c.state.S3Profiles[idx].Creds
	}

	var (
		bucket, url, region, accessKey, secretKey string
	)

	if existing != nil {
		bucket = existing.Bucket
		url = existing.Url
		region = existing.Region
		accessKey = existing.AccessKey
		secretKey = existing.SecretKey
	}

	if bucket, err = c.promptWithDefault("S3 bucket", bucket); err != nil {
		fmt.Printf("aborted: %v\n", err)
		return
	}
	if url, err = c.promptWithDefault("S3 endpoint URL", url); err != nil {
		fmt.Printf("aborted: %v\n", err)
		return
	}
	if region, err = c.promptWithDefault("S3 region", region); err != nil {
		fmt.Printf("aborted: %v\n", err)
		return
	}
	if accessKey, err = c.promptWithDefault("S3 access key", accessKey); err != nil {
		fmt.Printf("aborted: %v\n", err)
		return
	}
	if secretKey, err = c.promptWithDefault("S3 secret key", secretKey); err != nil {
		fmt.Printf("aborted: %v\n", err)
		return
	}

	profile := S3Profile{
		Name: name,
		Creds: types.S3Credentials{
			Bucket:    strings.TrimSpace(bucket),
			Url:       strings.TrimSpace(url),
			Region:    strings.TrimSpace(region),
			AccessKey: strings.TrimSpace(accessKey),
			SecretKey: strings.TrimSpace(secretKey),
		},
	}

	if idx >= 0 {
		c.state.S3Profiles[idx] = profile
		fmt.Printf("updated S3 profile '%s'\n", name)
	} else {
		c.state.S3Profiles = append(c.state.S3Profiles, profile)
		fmt.Printf("added S3 profile '%s'\n", name)
	}

	if err := c.saveState(); err != nil {
		fmt.Printf("failed to save config: %v\n", err)
	}
}

func (c *Cli) handleS3Delete(cmd *S3Delete) {
	if len(c.state.S3Profiles) == 0 {
		fmt.Println("no S3 profiles configured")
		return
	}

	name := strings.TrimSpace(cmd.Name)
	idx := -1

	if name != "" {
		idx = c.findS3ProfileIndex(name)
		if idx < 0 {
			fmt.Printf("no S3 profile named '%s'\n", name)
			return
		}
	} else {
		// interactive choose
		fmt.Println("S3 profiles:")
		for i, p := range c.state.S3Profiles {
			fmt.Printf("  %d) %s (bucket=%s, url=%s, access_key=%s)\n",
				i+1,
				p.Name,
				p.Creds.Bucket,
				p.Creds.Url,
				maskKey(p.Creds.AccessKey),
			)
		}

		in, err := c.readLine("Select profile to delete (name or number, empty = cancel): ")
		if err != nil {
			fmt.Printf("aborted: %v\n", err)
			return
		}
		in = strings.TrimSpace(in)
		if in == "" {
			fmt.Println("cancelled")
			return
		}

		// try number first
		if n, err := strconv.Atoi(in); err == nil {
			if n < 1 || n > len(c.state.S3Profiles) {
				fmt.Println("invalid selection")
				return
			}
			idx = n - 1
		} else {
			// treat as name
			idx = c.findS3ProfileIndex(in)
			if idx < 0 {
				fmt.Printf("no S3 profile named '%s'\n", in)
				return
			}
		}
		name = c.state.S3Profiles[idx].Name
	}

	confirm, err := c.readLine(fmt.Sprintf("Delete S3 profile '%s'? [y/N]: ", name))
	if err != nil {
		fmt.Printf("aborted: %v\n", err)
		return
	}
	if strings.ToLower(strings.TrimSpace(confirm)) != "y" && strings.ToLower(strings.TrimSpace(confirm)) != "yes" {
		fmt.Println("cancelled")
		return
	}

	// delete
	c.state.S3Profiles = append(c.state.S3Profiles[:idx], c.state.S3Profiles[idx+1:]...)
	fmt.Printf("deleted S3 profile '%s'\n", name)

	if err := c.saveState(); err != nil {
		fmt.Printf("failed to save config: %v\n", err)
	}
}
