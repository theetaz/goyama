-- 0002_graph.sql
-- Apache AGE graph initialization. The graph is a projection over the
-- relational tables maintained by a change-data-capture job. See docs/09.

BEGIN;

LOAD 'age';
SET search_path = ag_catalog, "$user", public;

SELECT create_graph('cropdoc');

-- Node labels
SELECT create_vlabel('cropdoc', 'Crop');
SELECT create_vlabel('cropdoc', 'Variety');
SELECT create_vlabel('cropdoc', 'AEZ');
SELECT create_vlabel('cropdoc', 'Disease');
SELECT create_vlabel('cropdoc', 'Pest');
SELECT create_vlabel('cropdoc', 'Remedy');
SELECT create_vlabel('cropdoc', 'Season');
SELECT create_vlabel('cropdoc', 'SoilGroup');

-- Edge labels
SELECT create_elabel('cropdoc', 'SUITABLE_IN');
SELECT create_elabel('cropdoc', 'AFFECTED_BY');
SELECT create_elabel('cropdoc', 'TREATED_BY');
SELECT create_elabel('cropdoc', 'RESISTANT_TO');
SELECT create_elabel('cropdoc', 'CONFUSED_WITH');
SELECT create_elabel('cropdoc', 'COMPANION_OF');
SELECT create_elabel('cropdoc', 'ROTATES_WITH');
SELECT create_elabel('cropdoc', 'GROWS_IN');
SELECT create_elabel('cropdoc', 'PREFERS_SOIL');
SELECT create_elabel('cropdoc', 'OF_CROP');

COMMIT;
