// ----------------------------------------------------------------------------
//
//     ***     AUTO GENERATED CODE    ***    Type: MMv1     ***
//
// ----------------------------------------------------------------------------
//
//     This file is automatically generated by Magic Modules and manual
//     changes will be clobbered when the file is regenerated.
//
//     Please read more about how to change this file in
//     .github/CONTRIBUTING.md.
//
// ----------------------------------------------------------------------------

package google

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceSecurityCenterNotificationConfig() *schema.Resource {
	return &schema.Resource{
		Create: resourceSecurityCenterNotificationConfigCreate,
		Read:   resourceSecurityCenterNotificationConfigRead,
		Update: resourceSecurityCenterNotificationConfigUpdate,
		Delete: resourceSecurityCenterNotificationConfigDelete,

		Importer: &schema.ResourceImporter{
			State: resourceSecurityCenterNotificationConfigImport,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(4 * time.Minute),
			Update: schema.DefaultTimeout(4 * time.Minute),
			Delete: schema.DefaultTimeout(4 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"config_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: `This must be unique within the organization.`,
			},
			"organization": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				Description: `The organization whose Cloud Security Command Center the Notification
Config lives in.`,
			},
			"pubsub_topic": {
				Type:     schema.TypeString,
				Required: true,
				Description: `The Pub/Sub topic to send notifications to. Its format is
"projects/[project_id]/topics/[topic]".`,
			},
			"streaming_config": {
				Type:        schema.TypeList,
				Required:    true,
				Description: `The config for triggering streaming-based notifications.`,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"filter": {
							Type:     schema.TypeString,
							Required: true,
							Description: `Expression that defines the filter to apply across create/update
events of assets or findings as specified by the event type. The
expression is a list of zero or more restrictions combined via
logical operators AND and OR. Parentheses are supported, and OR
has higher precedence than AND.

Restrictions have the form <field> <operator> <value> and may have
a - character in front of them to indicate negation. The fields
map to those defined in the corresponding resource.

The supported operators are:

* = for all value types.
* >, <, >=, <= for integer values.
* :, meaning substring matching, for strings.

The supported value types are:

* string literals in quotes.
* integer literals without quotes.
* boolean literals true and false without quotes.

See
[Filtering notifications](https://cloud.google.com/security-command-center/docs/how-to-api-filter-notifications)
for information on how to write a filter.`,
						},
					},
				},
			},
			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(0, 1024),
				Description:  `The description of the notification config (max of 1024 characters).`,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
				Description: `The resource name of this notification config, in the format
'organizations/{{organization}}/notificationConfigs/{{config_id}}'.`,
			},
			"service_account": {
				Type:     schema.TypeString,
				Computed: true,
				Description: `The service account that needs "pubsub.topics.publish" permission to
publish to the Pub/Sub topic.`,
			},
		},
		UseJSONNumber: true,
	}
}

func resourceSecurityCenterNotificationConfigCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	userAgent, err := generateUserAgentString(d, config.userAgent)
	if err != nil {
		return err
	}

	obj := make(map[string]interface{})
	descriptionProp, err := expandSecurityCenterNotificationConfigDescription(d.Get("description"), d, config)
	if err != nil {
		return err
	} else if v, ok := d.GetOkExists("description"); !isEmptyValue(reflect.ValueOf(descriptionProp)) && (ok || !reflect.DeepEqual(v, descriptionProp)) {
		obj["description"] = descriptionProp
	}
	pubsubTopicProp, err := expandSecurityCenterNotificationConfigPubsubTopic(d.Get("pubsub_topic"), d, config)
	if err != nil {
		return err
	} else if v, ok := d.GetOkExists("pubsub_topic"); !isEmptyValue(reflect.ValueOf(pubsubTopicProp)) && (ok || !reflect.DeepEqual(v, pubsubTopicProp)) {
		obj["pubsubTopic"] = pubsubTopicProp
	}
	streamingConfigProp, err := expandSecurityCenterNotificationConfigStreamingConfig(d.Get("streaming_config"), d, config)
	if err != nil {
		return err
	} else if v, ok := d.GetOkExists("streaming_config"); !isEmptyValue(reflect.ValueOf(streamingConfigProp)) && (ok || !reflect.DeepEqual(v, streamingConfigProp)) {
		obj["streamingConfig"] = streamingConfigProp
	}

	url, err := replaceVars(d, config, "{{SecurityCenterBasePath}}organizations/{{organization}}/notificationConfigs?configId={{config_id}}")
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Creating new NotificationConfig: %#v", obj)
	billingProject := ""

	// err == nil indicates that the billing_project value was found
	if bp, err := getBillingProject(d, config); err == nil {
		billingProject = bp
	}

	res, err := sendRequestWithTimeout(config, "POST", billingProject, url, userAgent, obj, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return fmt.Errorf("Error creating NotificationConfig: %s", err)
	}
	if err := d.Set("name", flattenSecurityCenterNotificationConfigName(res["name"], d, config)); err != nil {
		return fmt.Errorf(`Error setting computed identity field "name": %s`, err)
	}

	// Store the ID now
	id, err := replaceVars(d, config, "{{name}}")
	if err != nil {
		return fmt.Errorf("Error constructing id: %s", err)
	}
	d.SetId(id)

	// `name` is autogenerated from the api so needs to be set post-create
	name, ok := res["name"]
	if !ok {
		respBody, ok := res["response"]
		if !ok {
			return fmt.Errorf("Create response didn't contain critical fields. Create may not have succeeded.")
		}

		name, ok = respBody.(map[string]interface{})["name"]
		if !ok {
			return fmt.Errorf("Create response didn't contain critical fields. Create may not have succeeded.")
		}
	}
	if err := d.Set("name", name.(string)); err != nil {
		return fmt.Errorf("Error setting name: %s", err)
	}
	d.SetId(name.(string))

	log.Printf("[DEBUG] Finished creating NotificationConfig %q: %#v", d.Id(), res)

	return resourceSecurityCenterNotificationConfigRead(d, meta)
}

func resourceSecurityCenterNotificationConfigRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	userAgent, err := generateUserAgentString(d, config.userAgent)
	if err != nil {
		return err
	}

	url, err := replaceVars(d, config, "{{SecurityCenterBasePath}}{{name}}")
	if err != nil {
		return err
	}

	billingProject := ""

	// err == nil indicates that the billing_project value was found
	if bp, err := getBillingProject(d, config); err == nil {
		billingProject = bp
	}

	res, err := sendRequest(config, "GET", billingProject, url, userAgent, nil)
	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("SecurityCenterNotificationConfig %q", d.Id()))
	}

	if err := d.Set("name", flattenSecurityCenterNotificationConfigName(res["name"], d, config)); err != nil {
		return fmt.Errorf("Error reading NotificationConfig: %s", err)
	}
	if err := d.Set("description", flattenSecurityCenterNotificationConfigDescription(res["description"], d, config)); err != nil {
		return fmt.Errorf("Error reading NotificationConfig: %s", err)
	}
	if err := d.Set("pubsub_topic", flattenSecurityCenterNotificationConfigPubsubTopic(res["pubsubTopic"], d, config)); err != nil {
		return fmt.Errorf("Error reading NotificationConfig: %s", err)
	}
	if err := d.Set("service_account", flattenSecurityCenterNotificationConfigServiceAccount(res["serviceAccount"], d, config)); err != nil {
		return fmt.Errorf("Error reading NotificationConfig: %s", err)
	}
	if err := d.Set("streaming_config", flattenSecurityCenterNotificationConfigStreamingConfig(res["streamingConfig"], d, config)); err != nil {
		return fmt.Errorf("Error reading NotificationConfig: %s", err)
	}

	return nil
}

func resourceSecurityCenterNotificationConfigUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	userAgent, err := generateUserAgentString(d, config.userAgent)
	if err != nil {
		return err
	}

	billingProject := ""

	obj := make(map[string]interface{})
	descriptionProp, err := expandSecurityCenterNotificationConfigDescription(d.Get("description"), d, config)
	if err != nil {
		return err
	} else if v, ok := d.GetOkExists("description"); !isEmptyValue(reflect.ValueOf(v)) && (ok || !reflect.DeepEqual(v, descriptionProp)) {
		obj["description"] = descriptionProp
	}
	pubsubTopicProp, err := expandSecurityCenterNotificationConfigPubsubTopic(d.Get("pubsub_topic"), d, config)
	if err != nil {
		return err
	} else if v, ok := d.GetOkExists("pubsub_topic"); !isEmptyValue(reflect.ValueOf(v)) && (ok || !reflect.DeepEqual(v, pubsubTopicProp)) {
		obj["pubsubTopic"] = pubsubTopicProp
	}
	streamingConfigProp, err := expandSecurityCenterNotificationConfigStreamingConfig(d.Get("streaming_config"), d, config)
	if err != nil {
		return err
	} else if v, ok := d.GetOkExists("streaming_config"); !isEmptyValue(reflect.ValueOf(v)) && (ok || !reflect.DeepEqual(v, streamingConfigProp)) {
		obj["streamingConfig"] = streamingConfigProp
	}

	url, err := replaceVars(d, config, "{{SecurityCenterBasePath}}{{name}}")
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Updating NotificationConfig %q: %#v", d.Id(), obj)
	updateMask := []string{}

	if d.HasChange("description") {
		updateMask = append(updateMask, "description")
	}

	if d.HasChange("pubsub_topic") {
		updateMask = append(updateMask, "pubsubTopic")
	}

	if d.HasChange("streaming_config") {
		updateMask = append(updateMask, "streamingConfig.filter")
	}
	// updateMask is a URL parameter but not present in the schema, so replaceVars
	// won't set it
	url, err = addQueryParams(url, map[string]string{"updateMask": strings.Join(updateMask, ",")})
	if err != nil {
		return err
	}

	// err == nil indicates that the billing_project value was found
	if bp, err := getBillingProject(d, config); err == nil {
		billingProject = bp
	}

	res, err := sendRequestWithTimeout(config, "PATCH", billingProject, url, userAgent, obj, d.Timeout(schema.TimeoutUpdate))

	if err != nil {
		return fmt.Errorf("Error updating NotificationConfig %q: %s", d.Id(), err)
	} else {
		log.Printf("[DEBUG] Finished updating NotificationConfig %q: %#v", d.Id(), res)
	}

	return resourceSecurityCenterNotificationConfigRead(d, meta)
}

func resourceSecurityCenterNotificationConfigDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	userAgent, err := generateUserAgentString(d, config.userAgent)
	if err != nil {
		return err
	}

	billingProject := ""

	url, err := replaceVars(d, config, "{{SecurityCenterBasePath}}{{name}}")
	if err != nil {
		return err
	}

	var obj map[string]interface{}
	log.Printf("[DEBUG] Deleting NotificationConfig %q", d.Id())

	// err == nil indicates that the billing_project value was found
	if bp, err := getBillingProject(d, config); err == nil {
		billingProject = bp
	}

	res, err := sendRequestWithTimeout(config, "DELETE", billingProject, url, userAgent, obj, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return handleNotFoundError(err, d, "NotificationConfig")
	}

	log.Printf("[DEBUG] Finished deleting NotificationConfig %q: %#v", d.Id(), res)
	return nil
}

func resourceSecurityCenterNotificationConfigImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	config := meta.(*Config)

	// current import_formats can't import fields with forward slashes in their value
	if err := parseImportId([]string{"(?P<name>.+)"}, d, config); err != nil {
		return nil, err
	}

	stringParts := strings.Split(d.Get("name").(string), "/")
	if len(stringParts) != 4 {
		return nil, fmt.Errorf(
			"Saw %s when the name is expected to have shape %s",
			d.Get("name"),
			"organizations/{{organization}}/sources/{{source}}",
		)
	}

	if err := d.Set("organization", stringParts[1]); err != nil {
		return nil, fmt.Errorf("Error setting organization: %s", err)
	}
	return []*schema.ResourceData{d}, nil
}

func flattenSecurityCenterNotificationConfigName(v interface{}, d *schema.ResourceData, config *Config) interface{} {
	return v
}

func flattenSecurityCenterNotificationConfigDescription(v interface{}, d *schema.ResourceData, config *Config) interface{} {
	return v
}

func flattenSecurityCenterNotificationConfigPubsubTopic(v interface{}, d *schema.ResourceData, config *Config) interface{} {
	return v
}

func flattenSecurityCenterNotificationConfigServiceAccount(v interface{}, d *schema.ResourceData, config *Config) interface{} {
	return v
}

func flattenSecurityCenterNotificationConfigStreamingConfig(v interface{}, d *schema.ResourceData, config *Config) interface{} {
	if v == nil {
		return nil
	}
	original := v.(map[string]interface{})
	if len(original) == 0 {
		return nil
	}
	transformed := make(map[string]interface{})
	transformed["filter"] =
		flattenSecurityCenterNotificationConfigStreamingConfigFilter(original["filter"], d, config)
	return []interface{}{transformed}
}
func flattenSecurityCenterNotificationConfigStreamingConfigFilter(v interface{}, d *schema.ResourceData, config *Config) interface{} {
	return v
}

func expandSecurityCenterNotificationConfigDescription(v interface{}, d TerraformResourceData, config *Config) (interface{}, error) {
	return v, nil
}

func expandSecurityCenterNotificationConfigPubsubTopic(v interface{}, d TerraformResourceData, config *Config) (interface{}, error) {
	return v, nil
}

func expandSecurityCenterNotificationConfigStreamingConfig(v interface{}, d TerraformResourceData, config *Config) (interface{}, error) {
	l := v.([]interface{})
	if len(l) == 0 || l[0] == nil {
		return nil, nil
	}
	raw := l[0]
	original := raw.(map[string]interface{})
	transformed := make(map[string]interface{})

	transformedFilter, err := expandSecurityCenterNotificationConfigStreamingConfigFilter(original["filter"], d, config)
	if err != nil {
		return nil, err
	} else if val := reflect.ValueOf(transformedFilter); val.IsValid() && !isEmptyValue(val) {
		transformed["filter"] = transformedFilter
	}

	return transformed, nil
}

func expandSecurityCenterNotificationConfigStreamingConfigFilter(v interface{}, d TerraformResourceData, config *Config) (interface{}, error) {
	return v, nil
}