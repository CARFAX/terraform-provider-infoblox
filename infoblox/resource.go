package infoblox

import (
	"encoding/json"
	"github.com/CARFAX/skyinfoblox"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"reflect"
)

// CreateResource - Creates a new resource provided its resource schema
func CreateResource(name string, resource *schema.Resource, d *schema.ResourceData, m interface{}) error {

	attrs := GetAttrs(resource)
	obj := make(map[string]interface{})
	for _, attr := range attrs {
		key := attr.Name
		log.Println("Found attribute: ", key)
		updateInfoBloxObjectValue(key, attr, d, obj)
	}
	params := m.(map[string]interface{})
	client := params["ibxClient"].(*skyinfoblox.Client)

	log.Printf("Going to create an %s object: %+v", name, obj)
	ref, err := client.Create(name, obj)
	if err != nil {
		d.SetId("")
		return err
	}
	d.SetId(ref)
	return ReadResource(resource, d, m)
}

// ReadResource - Reads a resource provided its resource schema
func ReadResource(resource *schema.Resource, d *schema.ResourceData, m interface{}) error {

	params := m.(map[string]interface{})
	client := params["ibxClient"].(*skyinfoblox.Client)

	ref := d.Id()

	attrs := GetAttrs(resource)
	keys := []string{}
	for _, attr := range attrs {
		keys = append(keys, attr.Name)
	}
	// Read the data from Infoblox into obj
	obj := make(map[string]interface{})
	err := client.Read(ref, keys, &obj)
	if err != nil {
		d.SetId("")
		return err
	}
	//remove _ref from obj
	delete(obj, "_ref")
	for key := range obj {
		if isScalar(obj[key]) == true {
			log.Printf("Setting key %s to %+v\n", key, obj[key])
			d.Set(key, obj[key])
		} else if key == "extattrs" {
			/*
			   "extattrs": {
			     "Site":{
			       "value":"us-east-1"
			     }
			   }
			*/
			subMap := obj["extattrs"].(map[string]interface{})
			convertedToTerraform := make(map[string]interface{})
			for key, value := range subMap {
				//Extract the value in the "value" field from the response... @#$% infoblox api....
				//
				valueMap := value.(map[string]interface{})
				convertedToTerraform[key] = valueMap["value"].(string)
			}
			d.Set(key, convertedToTerraform)
		}
	}

	return nil
}

// DeleteResource - Deletes a resource
func DeleteResource(d *schema.ResourceData, m interface{}) error {

	params := m.(map[string]interface{})
	client := params["ibxClient"].(*skyinfoblox.Client)

	ref := d.Id()
	ref, err := client.Delete(ref)
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

// UpdateResource - Updates a resource provided its schema
func UpdateResource(resource *schema.Resource, d *schema.ResourceData, m interface{}) error {

	needsUpdate := false

	params := m.(map[string]interface{})
	client := params["ibxClient"].(*skyinfoblox.Client)

	ref := d.Id()
	attrs := GetAttrs(resource)
	obj := make(map[string]interface{})
	for _, attr := range attrs {
		key := attr.Name
		if d.HasChange(key) {
			updateInfoBloxObjectValue(key, attr, d, obj)
			log.Printf("Updating field %s, value: %+v\n", key, obj[key])
			needsUpdate = true
		}
	}

	log.Printf("UPDATE: going to update reference %s with obj: \n%+v\n", ref, obj)

	if needsUpdate {
		newRef, err := client.Update(ref, obj)
		if err != nil {
			log.Printf("Failed to update object... exiting")
			return err
		}
		d.SetId(newRef)
	}

	return ReadResource(resource, d, m)
}

func updateInfoBloxObjectValue(key string, attr ResourceAttr, d *schema.ResourceData, obj map[string]interface{}) {
	if key != "extattrs" {
		if v, ok := d.GetOk(key); ok {
			attr.Value = v
			obj[key] = GetValue(attr)
		}
	} else {
		//Key is extattrs.  Value attr.Value would be Cloud Region or Site or whatever the NEW key is.  Then have to get value from that new attr.  This is a work around due to limitations of submaps in terraform (e.g. those don't exist) and that InfoBlox API's are not designed particularly well.  They took a SOAP/XML based API and tried to use that for Restful services and it's danged hacky/ugly.
		if v, ok := d.GetOk(key); ok {
			jsonString, _ := json.Marshal(v)
			log.Println("Terraform in JSON for extattrs:", string(jsonString))
			//subMap SHOULD be a list of string elements
			infoBloxStupidAttrs := make(map[string]interface{})
			for key, value := range v.(map[string]interface{}) {
				//JSON: {"extattrs":{"value":{"Cloud Region":"us-east-1","Site":"AWS"}}}
				values := make(map[string]string)
				values["value"] = value.(string)
				infoBloxStupidAttrs[key] = values
			}
			obj["extattrs"] = infoBloxStupidAttrs
		}
	}
	jsonString, _ := json.Marshal(obj)
	log.Println("JSON results to be sent:", string(jsonString))
}

func isScalar(field interface{}) bool {
	t := reflect.TypeOf(field)
	if t == nil {
		return false
	}
	k := t.Kind()
	switch k {
	case reflect.Slice:
		return false
	case reflect.Map:
		return false
	}
	return true
}
