# Epic: Project Groups (Simplified)

## Overview
Global project groups that appear as tabs on the project list. Projects are assigned to a group via a dropdown in project settings (only shown if groups exist). No drag & drop.

## Implementation
- [x] Migration: `project_groups` table + `group_id` FK on projects
- [x] Backend: CRUD for groups + SetProjectGroup
- [x] Frontend: Tabs on project list with create/rename/delete
- [x] Frontend: Group dropdown in project settings (only if >1 group)
- [x] Filter projects by active tab
