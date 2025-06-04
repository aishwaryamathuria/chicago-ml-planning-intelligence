import geopandas as gpd
import pandas as pd

# Loading GeoJSONs into GeoDataFrames and ensuring consistent Coordinate Reference System.
community_areas = gpd.read_file("community_areas.geojson")
zip_codes = gpd.read_file("boundary_zipcode.geojson")
community_areas = community_areas.to_crs(epsg=3435)
zip_codes = zip_codes.to_crs(epsg=3435)

# Spatially intersecting the community area and ZIP code polygons to get exact overlaps.
intersections = gpd.overlay(community_areas, zip_codes, how='intersection')
intersections["intersection_area"] = intersections.geometry.area

#For each AREA_NUMBE, selecting the ZIP code with the largest intersection area.
dominant_zip = (
    intersections.loc[:, ["area_numbe", "zip", "intersection_area"]]
    .sort_values(by="intersection_area", ascending=False)
    .drop_duplicates(subset="area_numbe")
    .sort_values("area_numbe")
)
dominant_zip[["area_numbe", "zip"]].to_csv("community_area_to_zip_mapping.csv", index=False)
print(dominant_zip[["area_numbe", "zip"]])
