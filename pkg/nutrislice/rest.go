package nutrislice

import (
	"fmt"
	"time"

	"code.jhot.me/jhot/hats/internal/util"
	"github.com/go-resty/resty/v2"
)

type NutrisliceClient struct {
	restClient *resty.Client
	district   string
}

func New(district string) *NutrisliceClient {
	return &NutrisliceClient{
		restClient: resty.New().SetBaseURL(fmt.Sprintf("https://%s.api.nutrislice.com", district)),
		district:   district,
	}
}

func (c *NutrisliceClient) GetSchools() ([]School, error) {
	var data []School
	_, err := util.CheckSuccess(c.restClient.R().SetResult(&data).Get("menu/api/schools/?format=json"))
	return data, err
}

func (c *NutrisliceClient) GetWeeklyMenu(menuType MenuType, year int, month int, startDay int) (WeekMenu, error) {
	var data WeekMenu
	_, err := util.CheckSuccess(c.restClient.R().SetResult(&data).Get(fmt.Sprintf("menu/api/weeks/school/bauder/menu-type/%s/%d/%02d/%02d/?format=json", menuType, year, month, startDay)))
	return data, err
}

func (c *NutrisliceClient) GetTodaysMenu(menuType MenuType) ([]MenuItem, error) {
	now := time.Now()
	weekData, err := c.GetWeeklyMenu(menuType, now.Year(), int(now.Month()), now.Day())
	if err != nil {
		return []MenuItem{}, err
	}
	return weekData.Days[int(now.Weekday())].MenuItems, nil
}
