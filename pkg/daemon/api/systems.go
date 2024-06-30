package api

import (
	"net/http"

	"github.com/go-chi/render"
	"github.com/rs/zerolog/log"
	"github.com/wizzomafizzo/tapto/pkg/assets"
	"github.com/wizzomafizzo/tapto/pkg/database/gamesdb"
	"github.com/wizzomafizzo/tapto/pkg/platforms"
)

type System struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

type SystemsResponse struct {
	Systems []System `json:"systems"`
}

func (sr *SystemsResponse) Render(w http.ResponseWriter, _ *http.Request) error {
	return nil
}

func handleSystems(platform platforms.Platform) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Info().Msg("received systems request")

		indexed, err := gamesdb.IndexedSystems(platform)
		if err != nil {
			log.Error().Err(err).Msgf("error getting indexed systems")
			indexed = []string{}
		}

		if len(indexed) == 0 {
			log.Warn().Msg("no indexed systems found")
		}

		systems := make([]System, 0)

		for _, id := range indexed {
			sys, err := gamesdb.GetSystem(id)
			if err != nil {
				log.Error().Err(err).Msgf("error getting system: %s", id)
				continue
			}

			sr := System{
				Id: sys.Id,
			}

			sm, err := assets.GetSystemMetadata(id)
			if err != nil {
				log.Error().Err(err).Msgf("error getting system metadata: %s", id)
			}

			sr.Name = sm.Name
			sr.Category = sm.Category

			systems = append(systems, sr)
		}

		err = render.Render(w, r, &SystemsResponse{
			Systems: systems,
		})
		if err != nil {
			log.Error().Err(err).Msgf("error encoding systems response")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
