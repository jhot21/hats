package nutrislice

import "time"

// https://{{district-name}}.api.nutrislice.com/menu/api/schools/?format=json

type School struct {
	MonEnabled         bool   `json:"mon_enabled,omitempty"`
	MonStart           string `json:"mon_start,omitempty"`
	MonEnd             string `json:"mon_end,omitempty"`
	TueEnabled         bool   `json:"tue_enabled,omitempty"`
	TueStart           string `json:"tue_start,omitempty"`
	TueEnd             string `json:"tue_end,omitempty"`
	WedEnabled         bool   `json:"wed_enabled,omitempty"`
	WedStart           string `json:"wed_start,omitempty"`
	WedEnd             string `json:"wed_end,omitempty"`
	ThuEnabled         bool   `json:"thu_enabled,omitempty"`
	ThuStart           string `json:"thu_start,omitempty"`
	ThuEnd             string `json:"thu_end,omitempty"`
	FriEnabled         bool   `json:"fri_enabled,omitempty"`
	FriStart           string `json:"fri_start,omitempty"`
	FriEnd             string `json:"fri_end,omitempty"`
	SatEnabled         bool   `json:"sat_enabled,omitempty"`
	SatStart           string `json:"sat_start,omitempty"`
	SatEnd             string `json:"sat_end,omitempty"`
	SunEnabled         bool   `json:"sun_enabled,omitempty"`
	SunStart           string `json:"sun_start,omitempty"`
	SunEnd             string `json:"sun_end,omitempty"`
	ID                 int    `json:"id,omitempty"`
	Name               string `json:"name,omitempty"`
	Slug               string `json:"slug,omitempty"`
	DisplayCategoryIds []int  `json:"display_category_ids,omitempty"`
	ActiveMenuTypes    []struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Slug string `json:"slug,omitempty"`
		Urls struct {
			DigestMenuByDateAPIURLTemplate string `json:"digest_menu_by_date_api_url_template,omitempty"`
			DigestMenuByWeekAPIURLTemplate string `json:"digest_menu_by_week_api_url_template,omitempty"`
			FullMenuByDateAPIURLTemplate   string `json:"full_menu_by_date_api_url_template,omitempty"`
		} `json:"urls,omitempty"`
		FormatIsLocked bool   `json:"format_is_locked,omitempty"`
		MenuFormat     string `json:"menu_format,omitempty"`
	} `json:"active_menu_types,omitempty"`
	OrderingEnabled         bool   `json:"ordering_enabled,omitempty"`
	WayfindingNote          string `json:"wayfinding_note,omitempty"`
	PickupPeriods           []any  `json:"pickup_periods,omitempty"`
	ExternalID              any    `json:"external_id,omitempty"`
	Digest                  string `json:"digest,omitempty"`
	Logo                    string `json:"logo,omitempty"`
	Address                 string `json:"address,omitempty"`
	Timezone                any    `json:"timezone,omitempty"`
	Geolocation             any    `json:"geolocation,omitempty"`
	PhoneNumber             string `json:"phone_number,omitempty"`
	OperatingDaysByMenuType []any  `json:"operating_days_by_menu_type,omitempty"`
	OperatingStartDate      any    `json:"operating_start_date,omitempty"`
	OperatingEndDate        any    `json:"operating_end_date,omitempty"`
	Currency                struct {
		Code              string `json:"code,omitempty"`
		VisibleToCustomer bool   `json:"visible_to_customer,omitempty"`
	} `json:"currency,omitempty"`
}

// https://{{district-name}}.api.nutrislice.com/menu/api/weeks/school/{{school-slug}}/menu-type/{{lunch|breakfast}}/{{year}}/{{month}}/{{start-of-week}}/?format=json

type MenuItem struct {
	ID             int    `json:"id,omitempty"`
	Date           any    `json:"date,omitempty"`
	Position       int    `json:"position,omitempty"`
	IsSectionTitle bool   `json:"is_section_title,omitempty"`
	Bold           bool   `json:"bold,omitempty"`
	Featured       bool   `json:"featured,omitempty"`
	Text           string `json:"text,omitempty"`
	NoLineBreak    bool   `json:"no_line_break,omitempty"`
	BlankLine      bool   `json:"blank_line,omitempty"`
	Food           struct {
		ID                   int    `json:"id,omitempty"`
		Name                 string `json:"name,omitempty"`
		Description          string `json:"description,omitempty"`
		Subtext              string `json:"subtext,omitempty"`
		ImageURL             string `json:"image_url,omitempty"`
		HoverpicURL          string `json:"hoverpic_url,omitempty"`
		Price                any    `json:"price,omitempty"`
		Ingredients          string `json:"ingredients,omitempty"`
		FoodCategory         string `json:"food_category,omitempty"`
		FoodHighlightMessage struct {
			HeaderText string `json:"header_text,omitempty"`
			NormalText string `json:"normal_text,omitempty"`
			Color      string `json:"color,omitempty"`
		} `json:"food_highlight_message,omitempty"`
		FileURL              string `json:"file_url,omitempty"`
		DownloadLabel        string `json:"download_label,omitempty"`
		RoundedNutritionInfo struct {
			Calories      float64 `json:"calories,omitempty"`
			GFat          float64 `json:"g_fat,omitempty"`
			GSaturatedFat float64 `json:"g_saturated_fat,omitempty"`
			GTransFat     float64 `json:"g_trans_fat,omitempty"`
			MgCholesterol float64 `json:"mg_cholesterol,omitempty"`
			GCarbs        float64 `json:"g_carbs,omitempty"`
			GSugar        any     `json:"g_sugar,omitempty"`
			GAddedSugar   any     `json:"g_added_sugar,omitempty"`
			MgSodium      float64 `json:"mg_sodium,omitempty"`
			MgPotassium   any     `json:"mg_potassium,omitempty"`
			GFiber        float64 `json:"g_fiber,omitempty"`
			GProtein      float64 `json:"g_protein,omitempty"`
			MgIron        float64 `json:"mg_iron,omitempty"`
			MgCalcium     float64 `json:"mg_calcium,omitempty"`
			MgVitaminC    float64 `json:"mg_vitamin_c,omitempty"`
			IuVitaminA    float64 `json:"iu_vitamin_a,omitempty"`
			ReVitaminA    float64 `json:"re_vitamin_a,omitempty"`
			MgVitaminD    any     `json:"mg_vitamin_d,omitempty"`
		} `json:"rounded_nutrition_info,omitempty"`
		ServingSizeInfo struct {
			ServingSizeAmount string `json:"serving_size_amount,omitempty"`
			ServingSizeUnit   string `json:"serving_size_unit,omitempty"`
		} `json:"serving_size_info,omitempty"`
		HasNutritionInfo bool `json:"has_nutrition_info,omitempty"`
		Icons            struct {
			FoodIcons []struct {
				ID               int    `json:"id,omitempty"`
				SyncedName       string `json:"synced_name,omitempty"`
				Enabled          bool   `json:"enabled,omitempty"`
				SpotlightEnabled bool   `json:"spotlight_enabled,omitempty"`
				Sprite           struct {
					Slug      string `json:"slug,omitempty"`
					Name      string `json:"name,omitempty"`
					HelpText  string `json:"help_text,omitempty"`
					ClassName string `json:"class_name,omitempty"`
				} `json:"sprite,omitempty"`
				CustomIconURL          any    `json:"custom_icon_url,omitempty"`
				CustomIcon             any    `json:"custom_icon,omitempty"`
				CustomSpotlightIconURL any    `json:"custom_spotlight_icon_url,omitempty"`
				CustomSpotlightIcon    any    `json:"custom_spotlight_icon,omitempty"`
				Type                   int    `json:"type,omitempty"`
				Behavior               int    `json:"behavior,omitempty"`
				IsFilter               bool   `json:"is_filter,omitempty"`
				IsHighlight            bool   `json:"is_highlight,omitempty"`
				IsDigitalSignageOnly   bool   `json:"is_digital_signage_only,omitempty"`
				Slug                   string `json:"slug,omitempty"`
				IconSlug               string `json:"icon_slug,omitempty"`
				SpotlightIconSlug      any    `json:"spotlight_icon_slug,omitempty"`
				Name                   string `json:"name,omitempty"`
				SyncedName0            string `json:"_synced_name,omitempty"`
				HelpText               string `json:"help_text,omitempty"`
				SortOrder              int    `json:"sort_order,omitempty"`
				Enabled0               bool   `json:"_enabled,omitempty"`
				FoodIconGroup          int    `json:"food_icon_group,omitempty"`
			} `json:"food_icons,omitempty"`
			MyplateIcons []any `json:"myplate_icons,omitempty"`
		} `json:"icons,omitempty"`
		IconsApproved  bool  `json:"icons_approved,omitempty"`
		NestedFoods    []any `json:"nested_foods,omitempty"`
		AggregatedData struct {
			PriceAdjustment float64 `json:"price_adjustment,omitempty"`
			IconsApproved   bool    `json:"icons_approved,omitempty"`
			FoodIcons       []any   `json:"food_icons,omitempty"`
			Calories        float64 `json:"calories,omitempty"`
		} `json:"aggregated_data,omitempty"`
		OrderingEnabled    bool   `json:"ordering_enabled,omitempty"`
		FoodSizes          []any  `json:"food_sizes,omitempty"`
		DsCaloriesOverride string `json:"ds_calories_override,omitempty"`
		SyncedID           string `json:"synced_id,omitempty"`
		SyncPlaceholder    any    `json:"sync_placeholder,omitempty"`
		HasOptionsOrSides  bool   `json:"has_options_or_sides,omitempty"`
		Digest             string `json:"digest,omitempty"`
		PosItemID          string `json:"pos_item_id,omitempty"`
		SmartRecipeID      any    `json:"smart_recipe_id,omitempty"`
		HasSubfoods        bool   `json:"has_subfoods,omitempty"`
		MealPlanPrice      any    `json:"meal_plan_price,omitempty"`
		UseCustomSizes     bool   `json:"use_custom_sizes,omitempty"`
	} `json:"food,omitempty"`
	IsHoliday            bool    `json:"is_holiday,omitempty"`
	FoodList             any     `json:"food_list,omitempty"`
	StationID            any     `json:"station_id,omitempty"`
	IsStationHeader      bool    `json:"is_station_header,omitempty"`
	StationIsCollapsible bool    `json:"station_is_collapsible,omitempty"`
	Image                string  `json:"image,omitempty"`
	ImageDescription     any     `json:"image_description,omitempty"`
	ImageAlt             any     `json:"image_alt,omitempty"`
	ImageThumbnail       string  `json:"image_thumbnail,omitempty"`
	Category             string  `json:"category,omitempty"`
	Price                any     `json:"price,omitempty"`
	ServingSize          any     `json:"serving_size,omitempty"`
	ServingSizeAmount    float64 `json:"serving_size_amount,omitempty"`
	ServingSizeUnit      string  `json:"serving_size_unit,omitempty"`
	SmartRecipeID        any     `json:"smart_recipe_id,omitempty"`
	MenuID               int     `json:"menu_id,omitempty"`
	FoodVariationID      int     `json:"food_variation_id,omitempty"`
}

type WeekMenu struct {
	StartDate  string `json:"start_date,omitempty"`
	MenuTypeID int    `json:"menu_type_id,omitempty"`
	Days       []struct {
		Date                string     `json:"date,omitempty"`
		HasUnpublishedMenus bool       `json:"has_unpublished_menus,omitempty"`
		MenuItems           []MenuItem `json:"menu_items,omitempty"`
	} `json:"days,omitempty"`
	ID                    int       `json:"id,omitempty"`
	LastUpdated           time.Time `json:"last_updated,omitempty"`
	BoldAllEntreesEnabled bool      `json:"bold_all_entrees_enabled,omitempty"`
}

type MenuType string

const Breakfast MenuType = "breakfast"
const Lunch MenuType = "lunch"
