-- 0002_graph.sql
-- Apache AGE graph initialization. The graph is a projection over the
-- relational tables maintained by a change-data-capture job. See docs/09.

BEGIN;

LOAD 'age';
SET search_path = ag_catalog, "$user", public;

SELECT create_graph('goyama');

-- Node labels
SELECT create_vlabel('goyama', 'Crop');
SELECT create_vlabel('goyama', 'Variety');
SELECT create_vlabel('goyama', 'AEZ');
SELECT create_vlabel('goyama', 'Disease');
SELECT create_vlabel('goyama', 'Pest');
SELECT create_vlabel('goyama', 'Remedy');
SELECT create_vlabel('goyama', 'Season');
SELECT create_vlabel('goyama', 'SoilGroup');

-- Edge labels
SELECT create_elabel('goyama', 'SUITABLE_IN');
SELECT create_elabel('goyama', 'AFFECTED_BY');
SELECT create_elabel('goyama', 'TREATED_BY');
SELECT create_elabel('goyama', 'RESISTANT_TO');
SELECT create_elabel('goyama', 'CONFUSED_WITH');
SELECT create_elabel('goyama', 'COMPANION_OF');
SELECT create_elabel('goyama', 'ROTATES_WITH');
SELECT create_elabel('goyama', 'GROWS_IN');
SELECT create_elabel('goyama', 'PREFERS_SOIL');
SELECT create_elabel('goyama', 'OF_CROP');

COMMIT;
