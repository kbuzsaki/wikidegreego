/*
Implements a "Six degrees of separation" style search for the shortest "path"
between two wikipedia pages using a breadth first search.

The search treats the whole of wikipedia as a directed graph where each link
from a page A to another page B is treated as a directed edge from A to B.

For more info, see the wikipedia article:
https://en.wikipedia.org/wiki/Wikipedia:Six_degrees_of_Wikipedia

Here's an example of a (real!) shortest path from "bubble_gum" to "vladimir_putin"

bubble_gum
acetophenone
chemical_formula
nuclear_chemistry
vladimir_putin

This algorithm is pretty friendly to parallelization, though network
inconsistencies can make results inconsistent if multiple shortest paths
exist.

Its primary weakness is that it's a big memory hog for most searches that
require more than ~5 hops.
Hopefully iddfs.go will help with that :)
*/
package bfs

import (
	"fmt"
	api "github.com/kbuzsaki/wikidegree/api"
)

const frontierSize = 10 * 1000 * 1000
const numScraperThreads = 10

func FindNearestPathParallel(start string, end string) api.TitlePath {
	titles := make(chan string, frontierSize)
	pages := make(chan api.Page, 10)
	parsedPages := make(chan api.ParsedPage, 10)

	for i := 0; i < numScraperThreads; i++ {
		go loadPages(titles, pages)
	}
	go parsePages(pages, parsedPages)

	titles <- start
	visited := make(map[string]string)
	visited[start] = ""

	for parsedPage := range parsedPages {
		for _, link := range parsedPage.Links {
			if link == end {
				fmt.Println("Done!")
				fmt.Println()
				visited[link] = parsedPage.Title
				return pathFromVisited(visited, start, end)
			} else if len(visited[link]) == 0 {
				visited[link] = parsedPage.Title
				titles <- link
			}
		}
	}

	return nil
}

func loadPages(titles <-chan string, pages chan<- api.Page) {
	for title := range titles {
		fmt.Println("Loading:", title)
		if page, err := api.LoadPageContent(title); err == nil {
			pages <- page
		} else {
			fmt.Println("Failed to load: ", title)
		}
	}
}

func parsePages(pages <-chan api.Page, parsedPages chan<- api.ParsedPage) {
	for page := range pages {
		parsedPages <- api.ParsePage(page)
	}
}


type TitlePathQueue []api.TitlePath

func (pathQueue *TitlePathQueue) Push(titlePath api.TitlePath) {
	*pathQueue = append(*pathQueue, titlePath)
}

func (pathQueue *TitlePathQueue) Pop() api.TitlePath {
	var titlePath api.TitlePath
	titlePath, *pathQueue = (*pathQueue)[0], (*pathQueue)[1:]
	return titlePath
}

func FindNearestPathSerial(start string, end string) api.TitlePath {
	visited := make(map[string]bool)
	visited[start] = true
	frontier := TitlePathQueue{{start}}

	for len(frontier) > 0 {
		titlePath := frontier.Pop()

		fmt.Println("Loading:", titlePath)
		if page, err := api.LoadPageContent(titlePath.Head()); err == nil {
			parsedPage := api.ParsePage(page)

			for _, title := range parsedPage.Links {
				newTitlePath := titlePath.Catted(title)

				if title == end {
					return newTitlePath
				} else if !visited[title] {
					visited[title] = true
					frontier.Push(newTitlePath)
				}
			}
		} else {
			fmt.Println("Failed to load: ", titlePath.Head())
		}
	}

	return nil
}

func pathFromVisited(visited map[string]string, start string, end string) []string {
	// starts from the end of the graph and pops back
	var path []string
	parent := end
	for parent != start {
		path = append(path, parent)
		parent = visited[parent]
	}
	path = append(path, start)

	// reverse the path before returning
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	return path
}
