package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type Task struct {
	Id     int    `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

func ReadTask() []Task {
	file, err := ioutil.ReadFile("task.json")
	if err != nil {
		panic(err)
	}

	var data []Task
	err = json.Unmarshal(file, &data)
	if err != nil {
		panic(err)
	}

	return data
}

func ReadTaskDone() []Task {
	var data []Task

	file, err := ioutil.ReadFile("task.json")
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(file, &data)
	if err != nil {
		panic(err)
	}

	for _, v := range data {
		if v.Status == "Done" {
			return []Task{v}
		}
	}

	return data
}

func ReadTaskProgress() []Task {
	var data []Task

	file, err := ioutil.ReadFile("task.json")
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(file, &data)
	if err != nil {
		panic(err)
	}

	for _, v := range data {
		if v.Status == "Progress" {
			return []Task{v}
		}
	}

	return data
}

func UpdateTask(id int, task Task) string {
	data := ReadTask()
	data[id-1] = task
	file, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile("task.json", file, 0644)
	if err != nil {
		panic(err)
	}

	return "Task updated successfully"
}

func DeleteTask(id int) string {
	data := ReadTask()

	index := -1
	for i, v := range data {
		if v.Id == id {
			index = i
			data = append(data[:index], data[index+1:]...)
		}
	}

	if index == -1 {
		return fmt.Errorf("error: Task with id %d not found", id).Error()
	}

	file, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile("task.json", file, 0644)
	if err != nil {
		panic(err)
	}

	return "Task deleted successfully"
}

func AddTask(task Task) string {
	data := ReadTask()

	for _, v := range data {
		if task.Id == v.Id {
			return fmt.Errorf("error: Task with id %d already exists", task.Id).Error()
		}
	}

	data = append(data, task)
	file, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile("task.json", file, 0644)
	if err != nil {
		panic(err)
	}

	return "Task added successfully"
}

func main() {

	fmt.Println(DeleteTask(5))

	task := Task{
		Id:     7,
		Name:   "Buy Gundam",
		Status: "Progress",
	}

	fmt.Println(AddTask(task))

	// Read the tasks
	fmt.Println(ReadTask())

	// Read the tasks in Done
	fmt.Println(ReadTaskDone())

	// Read the tasks in Progress
	fmt.Println(ReadTaskProgress())
}
