-- +goose Up
-- +goose StatementBegin
INSERT INTO configs (id, game, name, author_nickname, is_global, user_id, config_body, created_at)
VALUES 
(
    'cs2-default-pro', 
    'cs2', 
    'Pro Player Base Settings', 
    'System', 
    TRUE, 
    0, 
    'sensitivity 1.2\nzoom_sensitivity_ratio 1.0\ncl_crosshairsize 2', 
    NOW()
),
(
    'cs2-alternative-fps', 
    'cs2', 
    'Max FPS Optimization', 
    'System', 
    TRUE, 
    0, 
    'mat_queue_mode 2\nr_drawtracers_firstperson 0\nr_dynamic 0', 
    NOW()
),
(
    'dota2-quickcast-standard', 
    'dota2', 
    'Standard Quick Cast Preset', 
    'System', 
    TRUE, 
    0, 
    '// === DOTA 2 STANDARD QUICK CAST PRESET ===\n\n// Глобальная активация быстрых кастов (Quick Casts)\ndota_ability_quick_cast 1\n\n// Умное применение заклинаний за пределами радиуса атаки\ndota_smart_target_out_of_range_cast 1\n\n// Умный выбор юнитов при множественном касте\ndota_player_smart_multiunit_cast 1\n\n// Добивание (денай) союзных крипов правой кнопкой мыши\ndota_force_right_click_attack 1\n\n// Все остальные внутриигровые настройки, графику и бинды клавиш оставляем стандартными', 
    NOW()
)
ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM configs WHERE id IN ('cs2-default-pro', 'cs2-alternative-fps', 'dota2-quickcast-standard');
-- +goose StatementEnd
