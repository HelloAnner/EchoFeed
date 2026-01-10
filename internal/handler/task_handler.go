package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/echofeed/echofeed/internal/model"
	"github.com/echofeed/echofeed/internal/scheduler"
	"github.com/echofeed/echofeed/internal/service"
)

// TaskHandler 任务API处理器
type TaskHandler struct {
	taskSvc *service.TaskService
	sched   *scheduler.Scheduler
}

// NewTaskHandler 创建任务API处理器
func NewTaskHandler(taskSvc *service.TaskService, sched *scheduler.Scheduler) *TaskHandler {
	return &TaskHandler{taskSvc: taskSvc, sched: sched}
}

// List 获取任务列表
func (h *TaskHandler) List(c *gin.Context) {
	tasks, err := h.taskSvc.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tasks)
}

// Get 获取单个任务
func (h *TaskHandler) Get(c *gin.Context) {
	id := c.Param("id")
	task, err := h.taskSvc.Get(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if task == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}
	c.JSON(http.StatusOK, task)
}

// Create 创建任务
func (h *TaskHandler) Create(c *gin.Context) {
	var task model.Task
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.taskSvc.Create(&task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 重新加载任务调度
	h.sched.ReloadTasks()

	c.JSON(http.StatusCreated, task)
}

// Update 更新任务
func (h *TaskHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var task model.Task
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	task.ID = id

	if err := h.taskSvc.Update(&task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 重新加载任务调度
	h.sched.ReloadTasks()

	c.JSON(http.StatusOK, task)
}

// Delete 删除任务
func (h *TaskHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.taskSvc.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 重新加载任务调度
	h.sched.ReloadTasks()

	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}

// Run 手动执行任务
func (h *TaskHandler) Run(c *gin.Context) {
	id := c.Param("id")
	date := c.Query("date") // 可选日期参数，格式: 2006-01-02
	h.sched.TriggerTaskRunForDate(id, date)
	c.JSON(http.StatusOK, gin.H{"message": "Task triggered", "date": date})
}
